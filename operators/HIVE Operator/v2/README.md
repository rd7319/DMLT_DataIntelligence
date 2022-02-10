HiveQuery

This operator provides functionality to query a Hive Metastore server using a HiveQL string and returns a response in the format of a delimited string.It supports Select, Insert, Show, Describe, Create, Drop, Truncate and Inserting records into table using data in the file.For each query the output is the result of the query in the form of string.
To insert data into table using the data records in the file, the file should be accessible to Read File operator. 

To run any query, select the Query_mode from the configuration parameter and give the query in the Query parameter except for query mode Insert_bulk.
Query parameter is mandatory for all query modes except insert_bulk
Query modes available: Select, insert_single, Show, Describe, Create, Drop, Truncate, and insert_bulk.

Note for Insert_Bulk:
Make sure the file and inserting table have same structure.
Change the table name in the script (Function definition: get_insert_bulk_query, line Number:66)
Provide the path of the input file in Read File Operator






