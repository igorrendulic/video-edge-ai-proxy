import { Logs, State } from "./StreamProcess";

export interface AppProcess {
    name?:string
    docker_user?:string,
    docker_repository?:string,
    docker_version?:string,
    env_vars?:VarPair[],
    mount?:VarPair[],
    port_mappings?:PortMap[],
    container_id?:string
    status?:string
    state?:State
    logs?:Logs
    created?:Number
    modified?:Number
    upgrade_available?:boolean
    newer_version?:string
    upgrading_now?:boolean
}

export interface VarPair {
    name:string,
    value:string
}

export interface PortMap {
    exposed:number,
    map_to:number,
}
