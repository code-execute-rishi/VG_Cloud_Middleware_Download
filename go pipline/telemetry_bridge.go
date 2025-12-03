func startTelemetryBridge(room *lksdk.Room) {
    node, _ := gomavlib.NewNode(gomavlib.NodeConf{
        Endpoints: []gomavlib.EndpointConf{
            gomavlib.EndpointUDPServer{Address: "0.0.0.0:14550"},
        },
        Dialect: common.Dialect,
    })
    
    for evt := range node.Events() {
        if frame, ok := evt.(*gomavlib.EventFrame); ok {
            
            switch msg := frame.Message.(type) {
            case *common.MessageAttitude:
                payload, _ := json.Marshal(map[string]interface{}{
                    "roll":  msg.Roll,
                    "pitch": msg.Pitch,
                    "yaw":   msg.Yaw,
                })

                room.LocalParticipant.PublishData(payload, livekit.DataPacket_LOSSY, []string{})
            }
        }
    }
}