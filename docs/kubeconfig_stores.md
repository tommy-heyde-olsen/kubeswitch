# Configure Kubeconfig stores

`kubeswitch` can recursively search over multiple directories, files and Kubeconfig stores.
If you neither provide a flag or a `SwitchConfig` file, it will default to the file `~/.kube/config`.

The `SwitchConfig` file is expected to be in the default location
on the local filesystem at `~/.kube/switch-config.yaml` or set via flag `--config-path`.
Example config files can be found [here](../resources/demo-config-files) but also in
the documentation for each store (see below).

Please check the documentation for each kubeconfig store on how to use it
 - [Filesystem](stores/filesystem/filesystem.md)
 - [Vault](stores/vault/use_vault_store.md)
 - [Gardener](stores/gardener/gardener.md)
 - [Rancher](stores/rancher/rancher.md)

Please note that, to search over **multiple** directories and kubeconfig stores,
you need to use the `SwitchConfig` file.

Also, if you do not have a default `~/.kube/config` file, kubeswitch will fail
with `Error: you need to point kubeswitch to a kubeconfig file`, so you need
to specify in the `SwitchConfig` file where to look.

## General Store configuration

These store configurations exists for each store.

### Specify the Kubeconfig name pattern to search for

The name or pattern to use when searching for kubeconfig files
within a store is specified via the command line flag `--kubeconfig-name`.

This can also be defined in the configuration file.
Either globally for each store, or per store as seen below.

```
$ cat ~/.kube/switch-config.yaml
kind: SwitchConfig
version: v1alpha1
kubeconfigName: "*.myconfig"
kubeconfigStores:
- kind: filesystem
  kubeconfigName: "*.myconfig"
  paths:
  - ~/.kube/my-other-kubeconfigs/
```

If you have several existing kubeconfig files in the directory `~/.kube`, the
following simple configuration may work for you:

```yaml
kind: SwitchConfig
version: v1alpha1
kubeconfigStores:
 - kind: filesystem
   kubeconfigName: "config*"
   paths:
     - ~/.kube
```

### Disable kubeconfig previews

Per default, kubeswitch shows a sanitized preview of the kubeconfig from the store.
This can however sometimes lead to high request rates & API throttling.
You can disable previews via the command line flag `--show-preview false` or the `SwitchConfig` file.

```
$ cat ~/.kube/switch-config.yaml

kind: SwitchConfig
version: v1alpha1
showPreview: false
```

### Optional stores

Optionally mark a store as not required via `required: false` to avoid logging errors when
the store is not reachable.
Useful when configuring a kubeconfig store that is not always available.
However, when searching on an index and wanting to retrieve the kubeconfig from an unavailable store,
it will throw an errors nonetheless.

```
kind: SwitchConfig
version: v1alpha1
kubeconfigStores:
- kind: filesystem
  required: false
  ...
```

### Disable prefixes for kubeconfig context names

Per default, each store prefixes discovered kubeconfig context names with a store-specific prefix.
This prefix usually contains semantic information (e.g., for the filesystem store the parent directory name or for the gardener store on which Seed the Shoot is placed).

However, you can choose to turn off the prefix.
This could make sense if you only got one kubeconfig file in `.kube/config`, hence a prefix of `.kube` is not necessary.

```
kind: SwitchConfig
version: "v1alpha1"
kubeconfigStores:
- kind: filesystem
  showPrefix: false
  paths:
  - "~/.kube/static-kubeconfigs/"
```

Please also note, that the kubeconfig files added with the CLI flag `--kubeconfig-path` as well as via Environment variable
`KUBECONFIG` never have a prefix.

## Advanced  Configurations

### Combined search over multiple stores

Just provide multiple store configurations leading to combined search results.

```
$ cat ~/.kube/switch-config.yaml
kind: SwitchConfig
version: v1alpha1
kubeconfigStores:
- kind: filesystem
  paths:
  - ~/.kube/my-other-kubeconfigs/
- kind: filesystem
  paths:
  - ~/.kube/my-next-kubeconfigs/
```

### Combined search over multiple stores with index

Only relevant if `kubeswitch` shall use an [index](search_index.md) for multiple kubeconfig stores of the
**same kind** that each use an index  (e.g., three filesystem stores that use an index).

This is useful when
- some search paths of a kubeconfig store (e.g., filesystem) should have a different index refresh interval
- there are multiple Vault instances or Gardener landscapes to read from.

A unique id is required to create a different index file per store, so that
the index of each store can be refreshed individually.

In the example below, two filesystem stores are configured.
Each store uses an index, because the field `refreshIndexAfter: 1h` is defined on the root level (see [here](search_index.md#enable-index-for-all-stores) for more information).
One store has the id `unique-1` and the other store has the `id unique-2`.
The Vault store uses the default id _default_ because it is the only Vault store that uses an index.

```
$ cat ~/.kube/switch-config.yaml

kind: SwitchConfig
version: v1alpha1
refreshIndexAfter: 1h
kubeconfigStores:
  - kind: filesystem
    id: unique-1
    paths:
    - "~/.kube/static-kubeconfigs/"
  - kind: filesystem
    id: unique-2
    paths:
    - "~/.kube/next-kubeconfigs/"
  - kind: Vault
    paths:
    - "path/in/vault"
```

## Using both CLI and `SwitchConfig` file

- The flag `--vault-api-address` takes precedence over the config field `vaultAPIAddress`.
- Specifying `--kubeconfig-path` and `--store` plus `kubeconfigPaths` in the config file
  causes a search over all of those paths combined.

# Additional considerations

To speed up the fuzzy search on the local filesystem,
it is recommended to put all the Kubeconfig files into a single directory containing only Kubeconfig files.
This is because the default `~/.kube` directory contains a bunch of other files
that have to be filtered out and thus slowing down the search.

To do this, create a `kubeswitch` alias via `--Kubeconfig-path` pointing
to this directory or setup the kubeconfig path in the `SwitchConfig`.
