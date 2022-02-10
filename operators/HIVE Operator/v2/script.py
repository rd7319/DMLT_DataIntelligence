import pandas as pd
from io import StringIO
from io import BytesIO
from subprocess import call
from hivejdbc import connect

#function to take action on input
def on_input(intrigger):
    hive_function(intrigger)

#function to extract first item from the list
def Extract(lst):
    return [item[0] for item in lst]
    
#Function to set up connection to Hive
def get_connection():
    conn = connect(host = api.config.hive_host,
                   port = api.config.hive_port,
                   database = api.config.database,
                   driver = '/usr/local/hive-jdbc-2.1.1-standalone.jar',
                   ssl = api.config.ssl_enabled,
                   trust_store = '/vrep/vflow/subengines/com/sap/python36/operators/ubp/com/sap/python36/hive/cm-auto-global_truststore.jks',
                   trust_password = '11XBYhbeufNalGo8hs0ZvZBXw8nNu6E3yxM3THkaYOO',
                   principal = api.config.principal,
                   krb5_conf = '/vrep/vflow/subengines/com/sap/python36/operators/ubp/com/sap/python36/hive/krb5.conf',
                   user_principal = api.config.user_principal,
                   user_keytab = '/vrep/vflow/subengines/com/sap/python36/operators/ubp/com/sap/python36/hive/dvmrmsvc01.headless.keytab')
    return conn
    
#Function to prepare query
def get_query():
    return api.config.Query
    
#Function to Prepare Insert query for bulk load from file data
def get_insert_bulk_query(intrigger):
    data = StringIO(intrigger.body.decode("utf-8"))
    df = pd.read_csv(data, index_col=False, dtype = 'str')
    
    #first element will be header and last element will be blank so not considering them when we conver Dataframe to csv
    df_1=df.to_csv(index=False).split('\n')[1:-1]
        
    #getting the column count for each file given
    column_count=int(len(str(df_1).split(','))/len(df_1))

    #generating data in necessary format
    data=''
    final_data=[]
    final_data1=[]
    for row in range(0,len(df_1)):
        for value in range(0,column_count):
            if value == 0:
                data=(str(df_1[row]).split(',')[value])
                data='('+data
                final_data.append(data)
            elif value == (column_count-1):
                data=(str(df_1[row]).split(',')[value])
                data=data+')'
                final_data.append(data)
            else:
                data=(str(df_1[row]).split(',')[value])
                final_data.append(data)
    final_data=str(final_data).replace("'(","('").replace(")'","')")   
    final_data1.append(final_data)
                
    #concatenate all the records into insert query
    insert_query = 'insert into table test5 values '+','.join(final_data1)
    insert_query=insert_query.replace('[','').replace(']','')
    
    return insert_query

#Function to get the result of the query
def get_result(query,cur):
        #Incase of queries - INSERT and CREATE, there is no resultset and hence seperating the logic
    if((query.split()[0] == "SELECT") or (query.split()[0] == "DESCRIBE") or (query.split()[0] == "SHOW")):
        
        #fetching the resulset recieved after query execution
        resultList = cur.fetchall()
        
        string = ""
        
        #attaching the header to the resultset - derived based on regular expression on DESCRIBE
        if(query.split()[0]=="SELECT"):
            if(int(query.count('SELECT *')) == 1 and int(query.count('JOIN')) < 1):
            
                #Fetching the tablename from the query inputtted
                tablename = query.split(" ")[3:][0]
                desc_query = "DESCRIBE" + " " +  tablename
            
                #executing the DESCRIBE to getch the column names
                cur.execute(desc_query)
                header = cur.fetchall()
                
                #attaching the header to the final dataset
                string = ','.join(map(str, Extract(header))) + "\n"
                
                #adding teh delimiter , to the resultset derived from the query inputted
                for x in resultList:
                    for y in x:
                        string = string + str(y) + "," ##api.config.delimiter ## Delimiter to separate Hive columns in output
                    string = string.rstrip(',') + "\n"
                
                #sending the ouput to the terminal    
                #api.send("output",string)
                
            #Incase the input query is select c1,c2,... statement - attaching the headers by parsing the query
            elif(int(query.count('SELECT *')) == 0 and int(query.count('JOIN')) < 1):
                #copy query by trimming spaces
                query_copy = "".join(query.split())
                #fetch the index of 'FROM' in the query
                index_from = query_copy.index("FROM")
                #fetch columns from query which will be between select and from 
                column_str = query_copy[6:index_from]
                #converting the column names in to list
                column_names = column_str.split(",")
                #attaching the header to the final dataset
                string = ','.join(column_names) + "\n"
                #adding teh delimiter , to the resultset derived from the query inputted
                for x in resultList:
                    for y in x:
                        string = string + str(y) + "," ##api.config.delimiter ## Delimiter to separate Hive columns in output
                    string = string.rstrip(',') + "\n"
            #Incase the input query is not a SELECT * or select c1,c2,... statement - we don't need to attach the headers    
            else:
                for x in resultList:
                    for y in x:
                        string = string + str(y) + "," ##api.config.delimiter ## Delimiter to separate Hive columns in output
                    string = string.rstrip(',') + "\n"
        #Incase of Show or describe query - no need of attaching the headers
        else:
            for x in resultList:
                    for y in x:
                        string = string + str(y) + "," ##api.config.delimiter ## Delimiter to separate Hive columns in output
                    string = string.rstrip(',') + "\n"
    #incase of the INSERT, CREATE, SET queries - since there is no resultset to be returned to output, program is sending a successful message.
    else:
        string = 'Executed successfully'
    
    return string
    
#Hive function to connect, query and process the data
def hive_function(intrigger):
    conn = get_connection()
    #Using Hive JDBC establishing the connection to the HIVE
    
    if api.config.Query_mode == 'insert_bulk':
        #prepare query
        inSql = get_insert_bulk_query(intrigger)
    else:
        inSql = get_query()
        
        
    #creating cursor to Hive connection
    cur = conn.cursor()
    
    #Using the cursor exceuting the query from the input
    cur.execute(inSql)
    
    #Casing the query for processing
    query = inSql.upper()
    
    api.send("output",get_result(query,cur))

if api.config.Query_mode == 'insert_bulk':
    api.set_port_callback("FileData", on_input)
else:
    api.set_port_callback("intrigger", on_input)