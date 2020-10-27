import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from 'src/environments/environment';
import { Observable } from 'rxjs';
import { StreamProcess } from '../models/StreamProcess';
import { Settings } from '../models/Settings';
import { ImageUpgrade, PullDockerResponse } from '../models/ImageUpgrade';

@Injectable({
  providedIn: 'root'
})
export class EdgeService {

  constructor(private http:HttpClient) { }

  listRTSP():Observable<[StreamProcess]> {
    return this.http.get<[StreamProcess]>(environment.LocalServerURL + "/api/v1/processlist");
  }

  getProcess(name:string):Observable<StreamProcess> {
    return this.http.get<StreamProcess>(environment.LocalServerURL + "/api/v1/process/" + name);
  }

  startProcess(process:StreamProcess) {
    return this.http.post<StreamProcess>(environment.LocalServerURL + "/api/v1/process", process);
  }

  stopProcess(name:string) {
    return this.http.delete(environment.LocalServerURL + "/api/v1/process/" + name);
  }

  getSettings():Observable<Settings> {
    return this.http.get<Settings>(environment.LocalServerURL + "/api/v1/settings");
  }

  overwriteSettings(settings:Settings) {
    return this.http.post<Settings>(environment.LocalServerURL + "/api/v1/settings", settings);
  }

  getDockerImages(tag:string) {
    return this.http.get<ImageUpgrade>(environment.LocalServerURL + "/api/v1/dockerimages?tag=" + tag);
  }

  pullDockerImage(tag:string,version:string) {
    return this.http.get<PullDockerResponse>(environment.LocalServerURL + "/api/v1/dockerpull?tag=" + tag + "&version=" + version);
  }

  getRTSPProcessUpgrades() {
    return this.http.get<[StreamProcess]>(environment.LocalServerURL + "/api/v1/processupgrades");
  }

  upgradeProcessContainer(process:StreamProcess) {
    return this.http.post<StreamProcess>(environment.LocalServerURL + "/api/v1/processupgrades", process);
  }
}
