#Script Monitoring Client

Script monitoring client can create a new heartbeat in OpsGenie and send a signal request; with a single command. Another command can disable or delete the heartbeat. 

You can use this plugin to monitor one-time jobs of your application using OpsGenie Heartbeat. You run the plugin with -action=start flag at the start of your script task. You specify a timetoexpire. Run the plugin again with -action=stop flag at the end. If the job doesn't end in the specified amount of time and so can't run the stop command,  OpsGenie will create an alert and notify you through voice call, e-mail, sms, mobile app etc. 

### Flags for usage:
**-apiKey** is mandatory. Use the API key of your OpsGenie Heartbeat integration.
**-name** is mandatory. You can use an existing hearbeat's name from [OpsGenie Heartbeats page](https://www.opsgenie.com/heartbeat/) or enter a new one for the plugin to add automatically.
**-action** is mandatory. Can be **start**, **stop** or **send**.
* **start** : Adds a new heartbeat to OpsGenie with the configuration from the given flags. If the heartbeat with the name specified in -name exists, updates the heartbeat accordingly and enables it. It also sends a heartbeat message to activate the heartbeat. 
* **stop** : Disables the heartbeat specified with -name, or deletes it if **-delete** is true. This can be used to end the heartbeat monitoring that was previously started.
* **send** : Sends a heartbeat message to reactivate the heartbeat specified with -name.

**-description** is optional. Sets the description of the heartbeat on [OpsGenie Heartbeats page](https://www.opsgenie.com/heartbeat/)
**-timetoexpire** is optional. Sets the expire time of the heartbeat that OpsGenie waits for a message request before creating an alert. The default value is 10.
**-intervalUnit** is optional. Can be **minutes**, **hours** or **days**. Default value is minutes. Sets the unit for the time to expire.
**-delete** is optional. Default value is false. Can be used to delete the heartbeat instead of disabling on **stop** commands.
