export interface Settings {
    name:string
    edge_key?:string
    edge_secret?:string
    gateway_id?:string
}

export interface DockerImageSummary {
    Containers?:[],
    Created:number,
    ID:string,
    Size?:number,
    RepoTags?:[string],
    Labels?:Map<string,string>,
    ParentID?:string,
}