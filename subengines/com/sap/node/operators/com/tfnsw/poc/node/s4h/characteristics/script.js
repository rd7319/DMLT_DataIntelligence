'use strict'

// get operator instance from Node SDK
const SDK = require("@sap/vflow-sub-node-sdk");
const op = SDK.Operator.getInstance();
const appLog = op.applicationLogger;
const { logger } = op;

const {connection, connectionType } = op.config;

logger.info(`connection ${JSON.stringify(connection)}`);
logger.info(`Connection Type: ${connectionType}`);
logger.info(`config: ${JSON.stringify(op.config)}`);

// Load S/4HANA Objects
const { ClfnCharacteristicForKeyDate,ClfnCharcDescForKeyDate,ClfnCharcValueForKeyDate,ClfnCharcValueDescForKeyDate } = require('@sap/cloud-sdk-op-vdm-characteristic-data-for-classification-service');
const { setTestDestination } = require('@sap-cloud-sdk/test-util');

// Set connection
setTestDestination({
    authentication: 'BasicAuthentication',
    name: 'myDestinationName',
    isTrustingAllCertificates: false,
    url: 'http://52.7.70.125:50000', //'https://tfnsw.demosap.com:44301',
    username: 'MDG_EXPERT',
    password: 'Welcome1'
});



op.getInPort("input").onMessage(async (message) => {
    op.logger.info(`Input received`);
    const attributes = message.Attributes;
    const body = message.Body;
    
    op.logger.info(body)
    const data = (body instanceof Array) ? message.Body : JSON.parse(message.Body);
    op.logger.info(data)
    op.logger.info(`Data Received`);
    const typ = typeof data;
    op.logger.info(typ);
    data.forEach( row => {
        op.logger.info(row)
        createChar(row);
    });

     
});


const createChar = async (charData) => {
    const {batch} = op.config;
    
    if (batch ) {
        op.logger.info(`Create Batch call for Characteristics`);

        
    } else {
        try {
            op.logger.info(`Create Single call for Characteristics`);
            //const characteristics = ClfnCharacteristicForKeyDate.builder()
            //     .characteristic(charData.Characteristic)
            //     .charcStatus(charData.CharcStatus)
            //     .charcDataType(charData.CharcDataType)
            //     .charcLength(charData.CharcLength)
            //     .toCharacteristicDesc([
            //         ClfnCharcDescForKeyDate.builder()
            //             .language('en')
            //             .charcDescription('DMLT CHARAC')
            //             .build()
            //     ])
            //     .build();
            
            op.logger.info(charData);
            const characteristics = ClfnCharacteristicForKeyDate.builder().fromJson(charData);
            
            op.logger.info(`Inside Service call`)    
            //op.logger.info(charData);
            op.logger.info(characteristics);
            
            //op.logger.info(`config: ${JSON.stringify(characteristics)}`);
            const newResp = await ClfnCharacteristicForKeyDate.requestBuilder()
                .create(characteristics)
                .execute({ destinationName: 'myDestinationName' })
                .catch(err => {
                    op.logger.error(`Error Code: ${err.code}`);
                    op.logger.error(`Error: ${err.message}`);
                    op.logger.error(`Cause: ${err.cause.message}`);
                    op.logger.error(`Root cause: ${err.rootCause.message}`)
                    
                    
                    op.getOutPort("error").send({
                        Attributes: { "message.request.id": "char" },
                        Encoding: 'string',
                        Body: {
                            "error": err.message || {},
                            "cause": err.cause.message || {},
                            "rootCause": err.rootCause.message || {}
                        }        
                    });
                 });
            
            if (newResp) {
                let message = {
                    Attributes: { "message.request.id": "class"  },
                    Encoding: 'string',
                    Body: newResp
                };
                op.getOutPort("output").send(message);
            }
       
        } catch (e) {
            op.logger.error(e);
            
            op.getOutPort("error").send({
                Attributes: {},
                Encoding: 'string',
                Body: e.message         
            });  
        }

    }
}