# file-client-task

### Task:
There is a simple implementation of file server based on http.FileServer handle (https://pkg.go.dev/net/http#example-FileServer).
The server instance is running on top of simple file folder which doesn’t have nested subfolders.
Please implement client which downloads files using this server.
You should download a file containing char 'A' on earlier position than other files.
In case several files have the 'A' char on the same the earliest position you should download all of them.
Each TCP connection is limited by speed. The total bandwidth is unlimited.
You can use any disk space for temporary files.
The goal is to minimize execution time and data size to be transferred.

Example:
If the folder contains the following files on server:
* 'file1' with contents: "---A---"
* 'file2' with contents: "--A------"
* 'file3' with contents: "-----------"
* 'file4' with contents: "--A----------"

then 'file2' and 'file4' should be downloaded

### My solution for the above task:
The client completes its execution with the below 3 steps:
1. Client retrieves the filenames from the given url
2. Client gets the files and the least index of the given character
3. Client stores the files in a zip file

To store the files there is a simple storage module with thread safety. Each file is stored in that storage.
To download the files a simple concurrent downloader has been implemented which takes a callback function
to handle the http response after the file is got. The downloader operates with a pub/sub mechanism.
The client initially subscribes to the downloader with the target url, the filenames and the callback function that mention above.
The callback function reads the file in chunks, finds the character's index in the content and stores those chunks in /tmp directory.
For this operation to be more optimized there is a thread safe integer where the found index is stored and only
the least is processed. While scanner traversing the chunks, if the current index passes the stored index the operation breaks.
Once the download has been completed, the client stores the files with the least index of the given character to a zip file.

### Usage:
* -char string := The files' matching character (default "A")
* -debug := Logger's debug flag (default true)
* -download-path string := The path that the files going to be stored (default "./")
* -max-concurrent-downloads uint := The number of maximum concurrent downloads (default 4)
* -url string := The requested file server's url (default "http://localhost:8080/")
