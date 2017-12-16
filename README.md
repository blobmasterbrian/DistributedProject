cdp332
bjq207

To build:
    go to src/webserver and src/backendserver and type "go build" in both folders
    make sure you have an empty data folder and empty log folder

To run:
    ./backendserver in src/backendserver
    ./webserver in src/webserver
To run replicas
    Do not run the webserver.
    follow steps to run backendserver in separate folder

How messages are sent between the web and data servers:
    A command request object is generated on the front end with proper parameters depending on 
    the user input.  The command request is then serialized using gob and sent over tcp to the
    backend.  The command request has an associated "Command Number" which dictates to the backend
    which function should be run.  The backend has a continuous loop to open connections and read
    command requests; a command response is sent back to the front end with a code detailing what
    the result of the command was.

How the structure of files is stored:
    Each file represents a single user.  Because usernames are unique, there is no potential conflict
    of having a 1 to 1 file user ratio.  Inside each file is the relivant User information that is
    loaded at server startup.  The file contains: username, password hash, users following the
    current user, users that the current user is following, and all of the posts of the user.
    The information serialized and stored on modification of the user using Gob.

How the locks work:
    There is a read/write lock on the global map storing the users, the only time a write lock is
    aquired is when a new user is created or a user is deleted, the rest of the operations are reads
    multiple reads can aquire the read lock at once, a write lock can only be held by one operation
    and during the time no reads can be made.   Each user also has a mutex for operations that modify
    or access and individual user's data.

How the replication works:
    On server statup, if there are no other active servers the newly started server is determined to be
    the master.  From then on every new server that is brought up queries the master for information about
    the filesystem and is given a unique id.  If the master dies, a bully-like algorithm is run, that chooses
    the next lowest ID num server to be the master.  If the next lowest server does not respond, then the servers
    will choose the lowest after that to be the new master.  The master is the only server that takes requests
    from the frontend.
    The frontend has an additional retry on sending infomration to the backend in the case that the master dies
    because replicas give the master 3 seconds to respond before determining the master to be dead.
    The master sends pings to all of the replicas to show that it is still alive.
    Expect some latency on the frontend when the master goes down to allow time for the election to occur.  The
    frontend should not error out.
    Replicas are currently only able to be hosted locally because we do not attempt to determine IP of the new replicas, only the port changes
    the requirement of local hosting can be easily modified and removed however, with a config file holding IPs of the replicas
    Frontend replication is also possible by running webserver in two seperate "replica folders", the hosted port would need to change to avoid port errors
