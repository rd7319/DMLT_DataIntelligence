Data Extraction using SLT to KAFKA
===========
#### Description
The Graph extracts data fom an ABAP table using SLT to the KAFKA and creates related messages.
Not mandatory, but for convenience, this example re-reads the data from the KAFKA feed and redirects it to the Terminal operator.

#### Prerequisites
1. You need a valid connection to the SAP LT Replication Server / SLT.
   Please create a valid and checked connection to an the SAP LT Replication Server / SLT system which supports the necessary SLT scenarios.
   Please use the related description of the SAP DI - Connection Management to implement and test such a connection.
   In this example the communication is based on HTTPS, but you may use other protocols such as HTTP, RFC or (WebSocket) WSRFC
2. You need a checked KAFKA connection and the ability to create, read and write messages to KAFKA.
   Please use the related description of the SAP DI - Connection Management to implement and test such a connection.
   In this example we used KAFKA as the consumer of the created messages.
3. A couple of Operators need to be in place: (ABAP) SLT Connector, ABAP Converter, Go Operator (–> String Operator, Determine Last Batch, Limit File Size, Check Last Batch), Python (--> splitting a portion of records to single records for messaging), Graph Terminator, KAFKA Producer and Consumer, Terminal, Wiretap
 

#### Configure and Run the Graph

1. In this example the SLT Connector replicates the specified ABAP table to KAFKA.
   The CDS Reader supports 3 ways to load data:
   a) the straight forward Initial Load, which copies the data from source to target but without any subsequent replication.
   b) the Replication, which includes the Initial Load and - based on a delta handling - the subsequent replication of changed data.
   c) the Delta Processing, which implies that no Initial Load takes place and changed data only will get processed.
2. The ABAP Converter feeds via a Message Converter into a Go Operator 'Determine Last Batch', which checks whether the SLT-based replication got stopped and is then launching a soft-kill (used at 'Check Last Batch')
3. The data keep flowing into KAFKA Producer using topic 'test_topic_SLT'.
4. The KAFKA Producer converts the records to messages within the configured parameters to KAFKA Topic 'test_topic_SLT'.
5. The Go Operator 'Check Last Batch' terminates the graph in case of the end of the replication; 
   without the two operators '* Last Batch', it can happen that the graph gets terminate before the last write got completed.
6. (optional) Simply to visualize the flow, the just used and created messages can be consumed by a KAFKA consumer and sent to Terminal/Wiretap.
7. Use <RUN> to execute this Graph.


8. (on top for better performance) Implementation of a Phyton script that splits the portion based table extraction into individual line items to support a single-message-based-on-single-data-record messaging.
   

#### Data Integration and Data Security
Please make yourself familiar with the related Integration Guides and the Security considerations as of
the SAP Data Hub "ABAP Integration Guide" https://help.sap.com/doc/61c7b7e293b74a45b50724c285df9560/2.7.latest/en-US/loio61c7b7e293b74a45b50724c285df9560.pdf ,
the SAP Data Hub "Administration Guide" https://help.sap.com/viewer/3f4043064eed446a895bc8ba7e61dc83/2.7.latest/en-US , and
SAP Note 2831756 "SAP Data Hub/ Data Intelligence ABAP Integration - Security Settings" 

<br>
<div class="footer">
   &copy; 2020 SAP SE or an SAP affiliate company. All rights reserved.
</div>
