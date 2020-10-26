export interface ImageUpgrade {
    name:string
    has_upgrade:boolean
    has_image:boolean
    current_version?:string
    highest_remote_version:string
    cam_type:string
}

export interface PullDockerResponse {
    response?:string
}