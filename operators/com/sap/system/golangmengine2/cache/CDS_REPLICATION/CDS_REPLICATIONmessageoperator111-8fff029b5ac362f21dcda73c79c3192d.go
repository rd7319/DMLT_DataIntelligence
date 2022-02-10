// Here the message header variable "ABAPfilenumber" gets calculated.
// "ABAPfilenumber" is used by the "Write File" operator in the subsequent step.
// By default, a new filenumber gets created after 1024kb - please adjust the code if necessary.

package main

import (
	"strconv"
)

//#### Begin of Configuration ####

// Initial Value for "ABAPfilenumber"
var mycounter=0    
// Threshold in kilobytes after which filenumber gets incremented
var mykblimit=1024 
//#### End of Configuration ####

var mysize=0
var GetInt func(string) int
var Output func(interface{})

func main() {

}


func Input(msg interface{}) {

    if msg == nil {
      return
    }

    mykblimit = GetInt("maxsizekb")
    
    if mykblimit == 0 { mykblimit = 1024 }

    mysize += len(msg.(map[string]interface{})["Body"].(string))
    if mysize >= 1024 * mykblimit {
      mycounter += 1
      mysize = len(msg.(map[string]interface{})["Body"].(string))
    }
     
    msg.(map[string]interface{})["Attributes"].(map[string]interface{})["ABAPfilenumber"] = strconv.Itoa(mycounter)

	Output(msg)
    
}
