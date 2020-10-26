export interface RTSP {
    device?:string
    username?:string
    password?:string
    route?:[string]
    address?:string
    port?:number
    route_found?:boolean
    available?:boolean
    authentication_type?:Number
}


export class GlobalVars {
    public static TempRTSPSearchResults:[RTSP];
    public static CameraTypes = new Map<string,string>([["rtsp","chryscloud/chrysedgeproxy"]]);
}