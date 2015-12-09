*** go upload server
- use kyoto tycoon for memcache

` docker run -d -p 1978:1978 hensansi/kyototycoon`

- edit main.go

`
        WEBSITE       = "http://192.168.99.100:8000/"  //your web server that host index.html
        HOST          = "192.168.99.100:8080"         //your go upload server host:port that start
        MEMCACHE      = "http://172.17.0.4:1978/"    //your kytotycoon server
`

- you can change upload server start port at function main default :8080

- edit ../../js/main.js

`
    $('#fileupload').fileupload({
        // Uncomment the following to send cross-domain cookies:
        //xhrFields: {withCredentials: true},
        url: '//192.168.99.100:8080/'    //your go upload server host:port that start
    });
`
