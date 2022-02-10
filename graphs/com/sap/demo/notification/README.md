Example Notification
===========

This graph serves as an example on how to use Notification and Pipeline operators from the Data Workflows category.

In the graph the Pipeline operator starts an internal example graph (com.sap.demo.counter) as subgraph. In case of the example graph terminates with status "dead", an email notification will be sent to the configured email address(es), see [Configurations](#configurations).

## Run the Graph

Follow the steps below to run Example Notification from the Modeler:

1. In the left panel, select the "Graphs" tab and navigate to "Examples > Example Notification".

2. In the tool bar, select "Run" (play button).

3. The "Status" panel indicates if the graph is running.

4. The example graph will appear in the "Status" panel and the status is running.

> To show the status of the example graph, please toggle on "Show subgraphs".

## Configurations

#### Configuring Pipeline operator (optional)

By default, the example graph is configured to terminate itself. In this context, the Data Workflow waits for the example graph to finish and will turn into "completed" after the example graph finishes. If you want to run an another graph and do not wait for the graph to finish, please follow the steps below:

1. Select "Pipeline" operator and open the configuration.

2. Go to "Graph Name" and select a graph from the list.

3. Go to "Running Permanently" and choose "true" from the list.

If you want to run a graph on a remote system. Please follow the documentation of the [Pipeline](../../../../../operators/com/sap/dh/vflowpipeline#configuration-parameters) operator.

#### Configuring Notification operator

1. Select "Notification" operator and open the configuration.

2. Go to "Connection", choose either "Configuration Manager" or "Manual". `When using "Configuration Manager", please create a connection via "Connection Management" first.`

3. Go to "Default From Value" and input an email address as sender.

4. Go to "Default To Value" and input one or more receiver email addresses.

5. Go to "Default Subject value" to give a subject to the email.

<br>
<div class="footer"> &copy; 2021 SAP SE or an SAP affiliate company. All rights reserved.</div>
