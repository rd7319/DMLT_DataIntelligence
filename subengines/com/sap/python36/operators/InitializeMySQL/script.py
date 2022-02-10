import json
def data_type(kind,abaptype,length,abaplen,decimals):
    if kind == 'C' or kind == 'N':
        return f'VARCHAR({length})'
    elif kind == 'I' or kind == 's':
        return abaptype
    elif kind == 'D':
        return 'DATE'
    elif kind == 'F':
        return f'DOUBLE({abaplen},{decimals})'
    elif kind == 'P':
        return f'DECIMAL({abaplen},{decimals})'
    elif kind == 'T':
        return 'TIME'
    else:
        return 'VARCHAR({length})'

def KEY_NULL_CHECK(KEY_FLAG,NULL_FLAG):
    if KEY_FLAG == 'X' or  NULL_FLAG == 'X':
        return 'NOT NULL'
    else:
      return ''

def PRIMARY_KEY_CHECK(FIELD_NAME,KEY_FLAG):
    if KEY_FLAG == 'X':
        return FIELD_NAME + ','
    else:
      return ''


def on_input(message):
    var = json.dumps(message.attributes) 
    A = json.loads(var)
    
    if A.get('message.batchIndex') > 1:
        api.send("out",message)
    else:    
        abaptypelist = A.get("metadata")
        dtypelist = A.get("ABAP").get("Fields")
            
        abtyplist = [i for i in abaptypelist if not (i.get('Field').get('ABAPTYPE') == '')]
        
        Field_Name_Append = ''
        Primary_Key_Append = ''
        for i,data in enumerate(abtyplist):
            #print(data.get('Field').get('COLUMNNAME') == dtypelist[i].get('Name'))
            data1 = data.get('Field')
            data2 = dtypelist[i]
            Field_Name_Append = Field_Name_Append + data1.get("COLUMNNAME") + ' ' +  data_type(data2.get('Kind'),data1.get('ABAPTYPE'),int(data1.get('OUTPUTLEN')),int(data1.get('ABAPLEN')), int(data1.get('DECIMALS'))) + ' ' + KEY_NULL_CHECK(data1.get("KEY"),data1.get("NULLABLE")) + ','
            Primary_Key_Append = Primary_Key_Append + PRIMARY_KEY_CHECK(data1.get("COLUMNNAME"),data1.get("KEY"))
            
        SQL_Code = 'CREATE TABLE IF NOT EXISTS ZIMRG (' + Field_Name_Append + 'PRIMARY KEY (' + Primary_Key_Append[:-1]+'));'   
        api.send("out",SQL_Code)

api.set_port_callback("input1", on_input)