export interface StreamProcess {
    name?:string
    image_tag?:string
    rtsp_endpoint?:string
    rtmp_endpoint?:string
    container_id?:string
    status?:string
    state?:State
    logs?:Logs
    created?:Number
    modified?:Number
    rtmp_stream_status?:RTMPStreamStatus
}

export interface State {
    Status?:string
    Running?:Boolean
    Paused?:Boolean
    Restarting?:Boolean
    OOMKilled?:Boolean
    Dead?:Boolean
    Pid?:Number
    ExitCode?:Number
    Error?:string
    StartedAt?:string
    FinishedAt?:string
}

export interface RTMPStreamStatus {
    streaming?:Boolean
    storing?:Boolean
}

export interface Logs {
    stdout?:string
    stderr?:string
}