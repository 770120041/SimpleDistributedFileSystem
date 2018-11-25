# sdfs for class cs425: Simple Distributed File System
## Usage
Use `go build` to compile the code 
Firstly need to start the introducer, use `./sdfs -isIntro` to start the introducer, introducer will use port 9123 
Then use `.sdfs -port portNumber` to start a host.

## Commands:

1. `id` to show its id(contains node IP address, port number and a timestampe when it started)
2. `membership` to show all the nodes in the system
3. `ping` to show all nodes it will ping 
4. `intro` to show who is now ther introducer
5. `master` to show the master for the file system
6. `put localname sdfaname` to put a file into SDFS file system
7. `get sdfsname localname` and `get-versions n sdfsname localname` to fetch a file from sdfs system
8. `store` to show which file is now stored in current host
9. `ls sdfsname` to show where a specific file is now being stored
10. `delete sdfsname` to delete a sdfsfile in SDFS file system
11. `showdb` in the master to show all the files stored in current database