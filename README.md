# Synq API

## Protos

The Synq API protos are present in the directory `./protos`. The generated docs for the same are available in `./docs`.

### Structure

The following is the expected folder structure for the protos.

```shell
protos
└── entity
    └── v1
        ├── entity.proto
        └── entity_service.proto
```

### Generating docs

Docs are generated automatically with CI pushes. If required, use the following command to generate docs locally. Note that this would still be overwritted with a CI push. 

```shell
bash generate_docs.sh
```

The script removes the existing docs in `./docs` and writes new ones.

> Prerequisites: [protoc](https://grpc.io/docs/protoc-installation/), [protoc-gen-doc](https://github.com/pseudomuto/protoc-gen-doc)
