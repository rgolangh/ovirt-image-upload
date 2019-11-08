package upload

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	ovirtsdk4 "github.com/ovirt/go-ovirt"
	"github.com/rgolangh/ovirt-image-upload/pkg/ovirt"
	log "github.com/sirupsen/logrus"
)

const BufferSize = 4 * 104857600 // 4MiB

func Upload(sourceUrl string, storageDomainId string) error {
	rand.Seed(time.Now().Unix())
	var sFile *os.File
	defer sFile.Close()
	if strings.HasPrefix(sourceUrl, "file://") || strings.HasPrefix(sourceUrl, "/") {
		// skip url download, its local
		local, err := os.Open(sourceUrl)
		if err != nil {
			return err
		}
		sFile = local
	} else {
		log.Infof("getting image from %v", sourceUrl)
		resp, err := http.Get(sourceUrl)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		sFile, err = ioutil.TempFile("/tmp", "*-ovirt-image-upload.downloaded")
		_, err = io.Copy(sFile, resp.Body)
		if err != nil {
			log.Errorf("problem with copy to tmp", err)
			return err
		}
		// remove it when done or cache? maybe --cache
		// defer os.Remove(sFile.Name())
	}
	getFileInfo, err := sFile.Stat()
	if err != nil {
		log.Errorf("the sourceUrl is unreachable %s", sourceUrl)
		return err
	}
	sFile, err = os.Open(sFile.Name())
	// gather details about the file
	uploadSize := getFileInfo.Size()
	header := make([]byte, 32)
	_, err = sFile.Read(header)
	if err != nil {
		log.Error("can't read header", err)
		return err
	}

	_, err = sFile.Seek(0, 0)
	if err != nil {
		log.Error("failed to seek file back", err)
		return err
	}

	log.Infof("get qcow header %v", header)
	qcowHeader, err := Parse(header)
	if err != nil {
		log.Errorf("failed to parse qcow header", err)
		return err
	}
	virtualSize := qcowHeader.Size
	log.Infof("upload size is %v", virtualSize)

	ovirtConf, err := ovirt.GetOvirtConfig()
	if err != nil {
		return err
	}
	conn, err := ovirtsdk4.NewConnectionBuilder().
		URL(ovirtConf.URL).
		Username(ovirtConf.Username).
		Password(ovirtConf.Password).
		CAFile(ovirtConf.CAFile).
		Insecure(ovirtConf.Insecure).
		Build()
	if err != nil {
		return err
	}
	defer conn.Close()

	// provisioned size must be the virtual size of the QCOW image
	// so must parse the virtual size from the disk. see the UI handling for that
	// for block format, the initial size of disk == uploadSize
	alias := "disk-upload-" + strconv.Itoa(rand.Int())
	diskBuilder := ovirtsdk4.NewDiskBuilder().
		Alias(alias).
		Format(ovirtsdk4.DISKFORMAT_COW).
		ProvisionedSize(int64(virtualSize)).
		InitialSize(int64(virtualSize)).
		StorageDomainsOfAny(
			ovirtsdk4.NewStorageDomainBuilder().
				Id(storageDomainId).
				MustBuild())
	diskBuilder.Sparse(true)
	disk, err := diskBuilder.Build()
	if err != nil {
		return err
	}

	addResp, err := conn.SystemService().DisksService().Add().Disk(disk).Send()
	if err != nil {
		log.Errorf("[DEBUG] Error creating the Disk (%s)", disk.MustName())
		return err
	}
	diskID := addResp.MustDisk().MustId()

	// Wait for disk is ready
	log.Debugf("[DEBUG] Disk (%s) is created and wait for ready (status is OK)", diskID)

	diskService := conn.SystemService().DisksService().DiskService(diskID)

	for {
		req, _ := diskService.Get().Send()
		if req.MustDisk().MustStatus() == ovirtsdk4.DISKSTATUS_OK {
			break
		}
		log.Info("waiting for disk to be OK")
		time.Sleep(time.Second * 5)
	}

	log.Infof("starting a transfer for disk id: %s", diskID)

	// initialize an image transfer
	imageTransfersService := conn.SystemService().ImageTransfersService()
	image := ovirtsdk4.NewImageBuilder().Id(diskID).MustBuild()
	log.Infof("the image to transfer: %s", alias)

	transfer := ovirtsdk4.NewImageTransferBuilder().Image(image).MustBuild()

	transferRes, err := imageTransfersService.Add().ImageTransfer(transfer).Send()
	if err != nil {
		log.Debugf("failed to initialize an image transfer for image (%v) : %s", transfer, err)
		return err
	}
	log.Infof("transfer response: %v", transferRes)

	transfer = transferRes.MustImageTransfer()
	transferService := imageTransfersService.ImageTransferService(transfer.MustId())
	for {
		req, _ := transferService.Get().Send()
		if req.MustImageTransfer().MustPhase() == ovirtsdk4.IMAGETRANSFERPHASE_TRANSFERRING {
			break
		}
		fmt.Println("waiting for transfer phase to reach transferring")
		time.Sleep(time.Second * 5)
	}

	//// plain get on the source image
	//download, err := http.Get(sourceUrl)
	//
	//if err != nil {
	//	log.Panicf("Problem in getting the source sourceUrl", sourceUrl)
	//}
	//defer download.Body.Close()

	uploadUrl, err := detectUploadUrl(transfer)
	if err != nil {
		return err
	}

	client := http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	urlReader := bufio.NewReaderSize(sFile, BufferSize)
	putRequest, err := http.NewRequest(http.MethodPut, uploadUrl, urlReader)
	putRequest.Header.Add("content-type", "application/octet-stream")
	putRequest.ContentLength = uploadSize

	if err != nil {
		log.Debugf("failed writing to create a PUT request %s", err)
		return err
	}

	_, err = client.Do(putRequest)
	if err != nil {
		return err
	}

	log.Info("finalizing...")
	_, err = transferService.Finalize().Send()

	if err != nil {
		return err
	}

	for {
		req, _ := diskService.Get().Send()

		// the system may remove the disk if it find it not compatible
		disk, ok := req.Disk()
		if !ok {
			return fmt.Errorf("the disk was removed, the upload is probably illegal")
		}
		if disk.MustStatus() == ovirtsdk4.DISKSTATUS_OK {
			break
		}
		log.Info("waiting for disk to be OK")
		time.Sleep(time.Second * 5)
	}

	return nil
}

func detectUploadUrl(transfer *ovirtsdk4.ImageTransfer) (string, error) {
	// hostUrl means a direct upload to an oVirt node
	insecureClient := http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	hostUrl, err := url.Parse(transfer.MustTransferUrl())
	if err == nil {
		optionsReq, err := http.NewRequest(http.MethodOptions, hostUrl.String(), strings.NewReader(""))
		res, err := insecureClient.Do(optionsReq)
		log.Infof("OPTIONS call on %s: %v", hostUrl.String(), res.StatusCode)
		if err == nil && res.StatusCode == 200 {
			return hostUrl.String(), nil
		}
		// can't reach the host url, try the proxy.
	}

	proxyUrl, err := url.Parse(transfer.MustProxyUrl())
	if err != nil {
		log.Errorf("failed to parse the proxy url (%s) : %s", transfer.MustProxyUrl(), err)
		return "", err
	}
	optionsReq, err := http.NewRequest(http.MethodOptions, proxyUrl.String(), strings.NewReader(""))
	res, err := insecureClient.Do(optionsReq)
	log.Infof("OPTIONS call on %s: %v", proxyUrl.String(), res.StatusCode)
	if err == nil && res.StatusCode == 200 {
		return proxyUrl.String(), nil
	}
	return "", err
}
