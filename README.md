# ovirt-image-upload

This is an experminal CLI tool to upload images from a URL into a storage domain.

### Usage
Create a credentials file under `~/.ovirt/ovirt-config.yaml`:

```
ovirt_url: https://enginefqdn/ovirt-engine/api
ovirt_username: admin@internal
ovirt_password: password
ovirt_ca: 
ovirt_insecure: true
```

Invoke the cli to upload [cirros](http://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img) image, for example:
```
bin/ovirt-image-upload \
   -s http://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img \
   -d 091c4f1e-5859-43d9-81b1-672b260f6912
```

The result is a disk under that storage domain.

### Build
```
make build
```
