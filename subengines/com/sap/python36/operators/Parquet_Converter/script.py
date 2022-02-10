## Custom code build in Base Python3 operator to append headers in csv file.
## Author: Shristi Drolia
##         Manish Kumar
## Publish Date: 
## Version : 1.0
# Import Python Libraries
import io
from io import StringIO
from io import BytesIO
import csv
import pandas as pd
import json
import numpy as np
import datetime
import time
import uuid


# read data coming from SLT operator
def on_input(inData):
# Formulation of date and time variable which will be used in filename
    mytimestamp = datetime.datetime.now()
    mydate = datetime.datetime.strftime(mytimestamp, "%Y%m%d")
    mytime = datetime.datetime.strftime(mytimestamp, "%H%M%S%f")
    mydatetime = mydate+"_"+mytime+"_"+str(uuid.uuid4())
    
# read body    
    data = StringIO(inData.body)
# read attribute    
    attr = inData.attributes
    
# columns extraction from json attributes    
    ABAPKEY = attr['ABAP']

# col variable to store column names
    col= []

# last batch determination 
    if('message.lastBatch' in attr):
        MSG = '1'
    else:
        MSG = '0'
# sending stop signal once last batch is detected    
    if(MSG == '1'):
        api.send("stop", "stop")
        
    else:    
# preparing list of columns         
        for columnname in ABAPKEY['Fields']:
            col.append(columnname['Name'])
# dataframe with columns and data            
        df = pd.read_csv(data, index_col=False, names=col, dtype = 'str')
# dataframe consversion to csv with header attributes
        df_csv = df.to_parquet()

#attribute to store date and time and counter 
        attr['mydatetime'] = mydatetime
        
        
# determine if the load is in initail phase or cdc and based on that folder name is determined
        if(df['IUUC_OPERATION'].iloc[0] in ('I', 'U', 'D')):
            attr['foldername'] = 'CDC_DI'
            

# sending csv as output            
            api.send("output", api.Message(attributes = attr, body=df_csv))
        
        else:
            attr['foldername'] = mydate
# sending csv as output            
            api.send("output", api.Message(attributes = attr, body=df_csv))
# reading next batch of data             
api.set_port_callback("input1", on_input)