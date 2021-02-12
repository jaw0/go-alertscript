// example


if( event.type == 'no' ){

    var r = web.post_json('https://api.example.com/testhook', null, {
        hash:  hex.encode( hash.sha1("hello world") ),
        event: event.type,
        key:  'abc123'
    })

    if( r.code != 200 ){
        console.log("POST FAILED: " + r.message)
    }

}

