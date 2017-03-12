# Simple Disk Volume Plugin for Docker

This plugin minimally extends docker's local disk mounting capabilities to
provide a sane way to handle the assignment of large spinning disks to docker
containers.

## Disk Structure
Due to the limitations on GPT metadata, `simple` depends on creating a small
(1mb) metadata partition at the start of each disk. This partition contains
a null teminated JSON blob of data which `simple` uses to inform disk assignment
choices.

## Query Language
`simple` is based on providing volume bindings via subdirectories, and using
the volume name in docker as a query language.

At a high level, `_` is used to separate settings and `.` is used to separate
key-values within settings i.e. `-v key.value_key2.value2:/volume`.

### Query Fields
Recognized query language fields are shown below.

* `label`
  Disk must have the given label. This field is required, and is used to 
  set a "type" for bound disks. Maximum length is 72 characters (GPT
  partition label length).

* `own-hostname`
  Disk metadata must contain the hostname of *this* host. Defaults to `false`.
  
* `own-machine-id`
  Disk metadata must contain the machine-id of *this* host. Defaults to `false`.
  
* `initialized`
  Disk must be initialized already. Do not initialize a new disk to satisfy the
  request.

* `basename`
  Specify a custom basename to use for disk mountpoints. Defaults to `simple-`.

* `naming-style`
  Specify the naming style to use for multiple disks. Options are "numeric"
  and "uuid". Default is `numeric`.

* `exclusive`
  Once a disk is assigned, do not assign it to any other containers requesting
  disk resources. Default is `true`.

* `min-size`
  Minimum disk size to consider.

* `max-size`
  Maximum disk size to consider.

* `min-disks`
  Minimum number of disks to add to the mount. Default `1`. Can be set to `0`
  in which case an empty, read-only mount is created immediately. This is only
  useful with `dynamic-mounts.true`.

* `max-disks`
  Maximum number of disks to add to the mount. Default `0` means unlimited.

* `dynamic-mounts`
  Monitor udev and dynamically add/remove devices. Defaults to `false`.

* `persist-numbering`
  Valid with "numeric" naming, and `max-disks` > 0 only. `simple` will keep
  the container directory populated with mount points that are read-only even
  if their are no disks to fill them.

* `filesystem`
  Disk must have the given filesystem type.
  
* `encryption-passphrase`
  If the disk is created, use this encryption passphrase. If the disk is matched
  it must be encrypted and usable with this passphrase.
  
* `encryption-cipher`
  If the disk is created, use this LUKS cipher string. Ignored on match (but
  logs a warning if different).

* `encryption-key-size`
  If the disk is created, use this LUKS key size. Ignored on match (but
  logs a warning if different).
  
* `encryption-hash`
  If the disk is created, use this LUKS hash string. Ignored on match (but
  logs a warning if different).

## simplectl
All of the operations simple performs supporting disk provisioning are made
available via the `simplectl` tool which can be used to prove out and test
different functions, as well as manually lifecycle provisioning operations.

For example to observe simple setting up an encrypted volume from a query
path:
```bash
$ dd if=/dev/zero of=test.img bs=1M count=0 seek=10000
$ losetup -f test.img
$ simplectl --log-level=debug initialize-disk /dev/loop0 \
    encryption-passphrase.yahFiepha9Cai9Iep1Baeb2ofeiKae_filesystem.ext4
```

## Life Cycle
When a docker container is launched with the volume driver, all local disks
are scanned for their `udev` data. Unpartitioned disks without filesystems on
them are by default considered candidates for assignment.

After simple has gathered as many disks as match the query, it will initialize
the actual volume mount it will pass to the container. This is achieved by
mounting a very small `tmpfs` (4k) to hold the volume mount directories. Each
disk is then mounted into a directory given the name `<label><label-style>`.

After the folders are created, the tmpfs is remounted read-only

## Automatic typing
simple will also take the designated "untyped" value for a partition
and add a different type to it (by changing the partition label). Type
changes by simple *always* destroy the data on the partition in order
to prevent information leakage between containers. Do them manually if
you want to migrate your setup.

# Future
* Automatic provisioning - simple will eventually be able to match and
  change filesystem types (implemented by standard unix commands).
 
