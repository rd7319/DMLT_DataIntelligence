import pandas as pd
import copy
import json
import io
import numpy as np

counter = 0
main_df = pd.DataFrame()
main_dict = {}
col_dict = {}

class Functor:
    def __init__(self, n): 
        self.port = n
    
    def get_tab_cols_keys(self,abtyplist,sid):
        keys = []
        cols = []
        global col_dict
        
        if not col_dict.get(sid):
            col_dict.update({sid : {}})
        #api.logger.info(col_dict)
        for i in abtyplist:
            name = i.get('Field').get('COLUMNNAME')
            name_sid = f"{i.get('Field').get('COLUMNNAME')}_{sid}"
            
            #if col_dict.get(sid):
            col_dict.get(sid).update({name : name_sid})
            
            if i.get('Field').get('KEY') == 'X':
                col_dict.get(sid).update({name : name})
                cols.append(name)
                keys.append(name)
            else:
                col_dict.get(sid).update({name : name_sid})
                cols.append(name_sid)
                # cols.append(i.get('Field').get('COLUMNNAME'))
        cols.append(f"Table_{sid}")
        cols.append(f"Flag_{sid}")
        #api.logger.info(col_dict)
        return cols,keys
    
    def on_input(self,msg):
        global counter
        global main_df
        global main_dict
        # global col_dict
        #counter += 1
        inp_ports = api.get_inport_names()
        #api.logger.info(counter)
        
        message = msg
        
        #Read attributes
        var = json.dumps(message.attributes) 
        attr = json.loads(var)    
        
        #CHECK If its last batch
        if "message.lastBatch" in attr:
            #api.logger.info(col_dict)
            counter += 1
            if counter == len(inp_ports):
                a = str(main_dict.get("JD1").shape)
                b = str(main_dict.get("JD2").shape)
                api.logger.info(a)
                api.logger.info(b)

                
                main_sys = inp_ports[0]
                main_dict[main_sys] = main_dict[main_sys].iloc[:,:-2]
                
                for n,df in enumerate(main_dict.values()):
                    # if n == 0:
                    #     continue
                    if list(main_dict.keys())[n] == main_sys:
                        continue
                    df = df.iloc[:,:-2]
                    main_df = main_dict[main_sys].join(df,how='inner')
                    
                api.logger.info(main_df.shape)    
                api.logger.info(f"{pd.__version__}")
                for i in inp_ports:
                    if i == main_sys:
                        api.logger.info(f"inside main sys{i}")
                        continue
                    
                    api.logger.info(col_dict[i])
                    
                    for data in col_dict[i]:
                        
                        col_sid = col_dict.get(i).get(data)
                        col_main = col_dict.get(main_sys).get(data)
                        
                        if data == col_sid:
                            continue               
                        api.logger.info(col_sid)
                        api.logger.info(col_main)
                        
                        main_df[col_sid] = main_df.apply(lambda x : x if x[col_sid] != x[col_main] else 
                                                              np.nan,axis=1)

                # df_list = list(main_dict.values())
                # main_df = pd.concat(df_list).drop_duplicates(keep=False)
                api.logger.info("out of df")
                api.logger.info(main_df.head())
                api.logger.info(main_df.shape)
                api.send("out",main_df.to_csv())    
            
        else:
            
            dtypelist = attr.get('ABAP').get('Fields')
            abaptypelist = attr.get('metadata')
            abtyplist = [i for i in abaptypelist if not (i.get('Field').get('ABAPTYPE') == '')]
            
            current_port = self.port
            #Read data to DataFrame
            data_stream = io.StringIO(message.body)
            
            #Get Cols and Keys from MySQL
            cols,keys = self.get_tab_cols_keys(abtyplist,current_port)
            
            temp_df = pd.read_csv(data_stream,names=cols,index_col=keys,dtype=str)
            
            # temp_df.insert(0,'SystemID',str(current_port))
            # keys.insert(0,'SystemID')
            
            # temp_df = temp_df.set_index(keys)            
            
            if current_port not in main_dict:
                main_dict[current_port] = copy.deepcopy(temp_df)
            else:
                main_dict[current_port] = main_dict[current_port].append(temp_df)
        
        
        # if counter == len(inp_ports):
        #     counter = 0
        

# inp_port = api.get_inport_names()
# api.logger.info(inp_port)

# for i in inp_port:
#     api.logger.info(i)
#     def data_in(data):
#         api.logger.info(f"Port is {i}")
#         on_input(data)
#     api.set_port_callback(i, data_in)

obj_dict = {}

inp_port = api.get_inport_names()
api.logger.info(inp_port)

for i in inp_port:
    obj_dict[i] = Functor(i)
    if obj_dict[i]:
        api.set_port_callback(i, obj_dict[i].on_input)