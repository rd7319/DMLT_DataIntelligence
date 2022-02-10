package main


func main() {

}

var Finished func(interface{})

func Input(msg interface{}) {

 if msg.(map[string]interface{})["Attributes"].(map[string]interface{})["message.lastBatch"] == true {
     if Finished != nil {
       Finished("true")
     }
 }
 

}