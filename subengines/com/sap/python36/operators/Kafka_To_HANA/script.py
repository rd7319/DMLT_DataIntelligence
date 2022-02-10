import pandas as pd
from io import StringIO
import csv
import json

def map_dtypes(meta,fields):
    name = meta['Field']['COLUMNNAME']
    cls = ''
    typ = {}
    prec = 0
    scale = 0
    for field in fields:
        if field['Name'] == name:
            kind = field['Kind']
            
            if kind == 'C':
                cls = 'string'
                typ['hana'] = 'NVARCHAR'
            elif kind == 'S' or kind == 'I':
                cls = 'integer'
                typ['hana'] = 'INTEGER'
            elif kind == 'P':
                cls = 'decimal'
                typ['hana'] = 'DECIMAL'
                prec = int(meta['Field']['ABAPLEN'].lstrip('0'))
                scale = field['Decimals']
            elif kind == 'F':
                cls = 'float'
            elif kind == 'D':
                cls = 'timestamp'
                typ['hana'] = 'DATE'
            elif kind == 'T':
                cls = 'timestamp'
                typ['hana'] = 'TIME'    
            else:
                cls = 'string'
                typ['hana'] = 'NVARCHAR'
    return cls,kind,typ,prec,scale    

def on_input(inData):
    tabname = str(api.config.TabName)
    # read body
    data = StringIO(inData.body) 
    # read attributes
    var = json.dumps(inData.attributes) 
    result = json.loads(var)
    # from here we start json parsing 
    if 'message.lastBatch' in result:
        #last batch
        api.logger.info("Last batch")
        
    else:
        
        ABAP = result['ABAP']
        Fields = ABAP['Fields']
        
        meta = result['metadata']
        
        tabmsg = {}
        tabmsg['Attributes'] = {}
        tabmsg['Attributes']['table'] = {}
        tabmsg['Attributes']['table']['version'] = 1
        
        tabmsg['Attributes']['table']['columns'] = []
        cols = []
        keys = []
        
        for field in meta:
            name = str(field['Field']['COLUMNNAME'])
            size = field['Field']['OUTPUTLEN'].lstrip('0')
            if size == '':
                continue
            else:
                size = int(size)
            
            nullable = "False"
            if field['Field']['NULLABLE'] == 'X':
                nullable = "True"
                
            cls,kind,typ,prec,scale = map_dtypes(field,Fields)     
            
            if cls == 'string':    
                cols.append({'name':name,'class':cls,'type':typ,'precision':prec,'scale':scale,'size':size,'nullable' : nullable})
            elif cls =='timestamp':
                cols.append({'name':name,'class':cls,'type':typ,'nullable' : nullable})
            elif cls == 'float':
                cols.append({'name':name,'class':cls,'type':typ,'nullable' : nullable})
            else:
                cols.append({'name':name,'class':cls,'type':typ,'precision':prec,'scale':scale,'nullable' : nullable})
            
            if field['Field']['KEY'] == 'X':
                keys.append(name)
                
        tabmsg['Attributes']['table']['primaryKey'] = keys
        tabmsg['Attributes']['table']['columns'] = cols
        tabmsg['Encoding'] = 'table'
        
        body = []
        tab_payload = ''
        while True:
        # Read one line.
            line = data.readline()
        # If reach the end of the data then jump out of the loop.
            if line == '':
                break
            else:
                lista = []
                if tab_payload == '':
                    tab_paylaod = line.split(',')[-2]
                    break;
                    
                #lista = line.split(',')
                #lista.append(line.strip())
                #body.append(lista)
        
        #tabmsg['Body'] = body
        
        #tabjson = json.dumps(tabmsg)
        
        if tabname == '':
            tabmsg['Attributes']['table']['name'] = tab_payload.format(inData.attributes["ABAP"]["Kind"])
        else:    
            tabmsg['Attributes']['table']['name'] = tabname.format(inData.attributes["ABAP"]["Kind"])
        
        api.send('out', api.Message(attributes=tabmsg['Attributes'], body=None))
        api.send('out2', api.Message(attributes=tabmsg['Attributes'], body=inData.body))
        
api.set_port_callback('input1', on_input)