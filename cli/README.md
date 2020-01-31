# Node Command Line Interface

## Manager
non-interactive mode
```sh
$ ./node  manager [command] [command options]
```

## Console
interactive mode
```sh
$ ./node  console
>>> [command] [command options]
```


### Global Options
```text
--help, -h     show help (default: false)
--version, -v  print the version (default: false)
```

### Commands
```text
dial_peer, dp     connect a new peer
prune_blocks, pb  delete block information
status, s         display the current status of the blockchain
net_info, ni      display network data
exit, e           exit
help, h           Shows a list of commands or help for one command
```

#### dial_peer
connect a new peer
```text
OPTIONS:
   --address value, -a value  id@ip:port
   --persistent, -p           (default: false)
   --help, -h                 show help (default: false)
```

#### prune_blocks
delete block information
```text
OPTIONS:
   --from value, -f value  (default: 0)
   --to value, -t value    (default: 0)
   --help, -h              show help (default: false)
```

#### status
display the current status of the blockchain
```text
OPTIONS:
   --json, -j  echo in json format (default: false)
   --help, -h  show help (default: false)
```

#### net_info
display network data
````text
OPTIONS:
   --json, -j  echo in json format (default: false)
   --help, -h  show help (default: false)
````

#### Small talk
- Sergey Klimov ([@klim0v](https://github.com/klim0v)): [Workshops MDD Dec'19: Node Command Line Interface](http://minter.link/p3)
