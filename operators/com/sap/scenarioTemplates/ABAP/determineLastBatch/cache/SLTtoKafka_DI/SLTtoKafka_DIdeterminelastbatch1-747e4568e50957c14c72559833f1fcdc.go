package main

func main() {

}
var CopyMessage func(map[string]interface{}) (map[string]interface{}, error)
var Out func(interface{})

func In(msg interface{}) {

    if ( msg.(map[string]interface{}) == nil ) || ( msg.(map[string]interface{})["Body"] == nil ) {
      return
    }

    m, err := CopyMessage(msg.(map[string]interface{}))
    if err != nil {
        panic(err)
    }

    //finish if empty string is sent
    if ( msg.(map[string]interface{})["Body"].(string) == "" ) ||
       ( msg.(map[string]interface{})["Body"].(string)[:1] == "\n" ) ||
       ( msg.(map[string]interface{})["Body"].(string) == "{\"DATA\":\"\"}" ) ||
       ( msg.(map[string]interface{})["Body"].(string)[1:] == 
         "<?xml version=\"1.0\" encoding=\"utf-16\"?><asx:abap xmlns:asx=\"http://www.sap.com/abapxml\" version=\"1.0\"><asx:values><DATA/></asx:values></asx:abap>" ) {
     m["Attributes"].(map[string]interface{})["message.lastBatch"] = "true"
     m["Attributes"].(map[string]interface{})["ABAPlastBatch"] = "true"
    } else {
     m["Attributes"].(map[string]interface{})["ABAPlastBatch"] = "false"
    }

  	msg.(map[string]interface{})["Attributes"] = m["Attributes"]
    
	Out(msg)
    
}
