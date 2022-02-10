package main

func main() {

}

var Finished func(interface{})
var Out func(interface{})

func In(msg interface{}) {
    
 if msg.(map[string]interface{})["Attributes"].(map[string]interface{})["message.lastBatch"] == "true" {
     if Finished != nil {
       Finished("true")
     }
 }
 

}