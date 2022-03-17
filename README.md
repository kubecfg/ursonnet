# Build

```bash
go install github.com/mkmik/ursonnet@latest
```


# Demo:

```console
$ ursonnet testdata/child.jsonnet '$.deployment.spec.template.spec.containers[0].resources.limits.cpu'   
testdata/common.libsonnet:27 
testdata/base.jsonnet:5 
testdata/common.libsonnet:23 
testdata/common.libsonnet:22 
testdata/config.libsonnet:5
```