# sdfs: Simple Distributed File System
## Usage
Use `go build` to compile the code 
The binary will be generated as "sdfs"(Macos) or "sdfs.exe"
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

## Implementation details
#### distribued membership protol
To implementing a distribued file system, we fistly need a distribute group protocol thus we will be able to know which node is currently in this group, and when there is a failure how can we manage the file storage.

I have designed a distributed membership protocol. In this protocol, there are many kinds of message used to maintain the functionality of this group. Like "JOIN" message and "JOINACK" message used for joining this group. Like "DELETE" or "UPDATA" used to update the membership, like "Ping" and "ACK" used to check if a node is still alive. They are all defined in package com, in "com.go" file.

For my system, there is a introducer, and other nodes are just hosts who can join the group. Each host is uniquely identified by its port number, IP address and a timestamp. I denoted it as NodeID. And when a node wants to join the group, It will send "Join" message to the introducer, and the introducer will reply with a "JoinACK" message and the pinglist for this node. 

The usage of pinglist is used to constantly check if a node is still alive. For each node, it will send "Ping" message to nods in its pinglist, if it didn't respond within a given condition, it will deem this node as dead and thus tell other nodes that some node dead. To make this system scalable, every node will not ping every one, they will only ping 4 nodes each time, thus my system can bear failures less than 4 node.


#### distribuetd file system
In my implementation, there will be a master who stores the meta info of the files in this system. This master is elected using the ring-based election algorithm. And the node with highest id will be elected as the master. When a master is elected, it will tell all the nodes that he is the master. And when any nods detected that the master fails, it will restart and election and elect a new master.

In my distribued file system, each file will have 4 replicas in our system. And when host Ause "put localname sdfsname", the node will tell master that it want to store a file by sending "STOREREQUEST" message to the master. The master will select 4 nodes and tell host A to send this file to those four nodes. The master never knows what's stored in this file and it was not necessarily stored in masster. But master will maintain a simple database which stores where files are stored, and how many versions there are for a specific file(Different versions means that a file with same sdfsname is stored in this system for more than once)

When we use "get sdfsname", the Host B will send a "GETREQUEST" message to the master, then master will tell host B a node who stores this file, then host B can download this file from this node. GET will defaulty send the newest version of the file. But we can also use "get-versions" to specify how many versions we want.

If some new node joins, master will check if some files have less than 4 replices, if so, it will tell some hosts to send a file to this new node to reach the state of 4 replices.

If some node files, master will check if some files have less than 4 replices, if so, it will try to send this file to different host to reach 4 replics.

If the master dies, no new files can be stored in this system, and a new election will be raised. After the election complete, the new master will use "Askfile" to ask each node what it stored in its file system. And using this info to reconstruct a file meta info database.


##Test
I have tested this program in my own computer, the files are stored in "tmp/sdfs/" directory, and also test on 10 machines in a VM. 