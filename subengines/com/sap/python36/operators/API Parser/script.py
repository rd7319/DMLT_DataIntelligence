import pandas as pd
from io import BytesIO
import csv
import json
def on_input(inData):
    # read body
    #data = BytesIO(inData.attributes)
    data = str(inData.body, encoding='utf-8')
    
    # read attributes
    var = json.dumps(data) 
    result = json.loads(json.loads(var))
    if 'containers' in result:
        for i in result['containers']:
            api.logger.info(i['name'])
        
api.set_port_callback('input1', on_input)