import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from 'src/environments/environment';
import { Observable } from 'rxjs';
import { StreamProcess } from '../models/StreamProcess';
import { DockerImageSummary, Settings } from '../models/Settings';
import { ImageUpgrade, PullDockerResponse } from '../models/ImageUpgrade';
import { AppProcess } from '../models/AppProcess';

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

  // camera/device docker images with possible upgrades
  getDeviceDockerImages(tag:string) {
    return this.http.get<ImageUpgrade>(environment.LocalServerURL + "/api/v1/devicedockerimages?tag=" + tag);
  }

  getAllDockerImages() {
    return this.http.get<[DockerImageSummary]>(environment.LocalServerURL + "/api/v1/alldockerimages")
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

  installApp(app:AppProcess) {
    return this.http.post<AppProcess>(environment.LocalServerURL + "/api/v1/appprocess", app);
  }

  removeApp(name:string) {
    return this.http.delete(environment.LocalServerURL + "/api/v1/appprocess/" + name);
  }

  listApps() {
    return this.http.get<[AppProcess]>(environment.LocalServerURL + "/api/v1/appprocesslist");
  }

  getApp(name:string) {
    return this.http.get<AppProcess>(environment.LocalServerURL + "/api/v1/appprocess/" + name);
  }
}
