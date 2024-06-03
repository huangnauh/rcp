Remote Copy CLI for houdeyun

Usage:
```
  hcp [flags]
```

Examples:
```
hcp SOURCE DEST
```

Flags:
```
  -h, --help           help for hcp
      --parallel int   The number of file transfers to run in parallel. (default 6)
  -v, --verbose        verbose output
```


Config:
```
/root/.config/hcp/hcp.yaml
```

```
remotes:
    - name: online                # rclone remote name
      bucket: huangnauh           # rclone remote bucket name
      mountpoint: /online
    - name: online
      bucket: huangnauh
      mountpoint: /online1

```