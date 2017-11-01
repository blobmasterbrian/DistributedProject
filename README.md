cdp332
bjq207

To build:
    go to src/webserver and src/backendserver and type "go build" in both folders
    make sure you have an empty data folder and empty lib folder

To run:
    ./backendserver in src/backendserver
    ./webserver in src/webserver

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

There are no modifications to the User Interface except a addtional check for username and password
length to be above 1.
