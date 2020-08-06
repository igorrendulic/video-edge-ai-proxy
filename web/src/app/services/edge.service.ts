import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from 'src/environments/environment';
import { Observable } from 'rxjs';
import { StreamProcess } from '../models/StreamProcess';
import { RTSP } from '../models/RTSP';

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

  rtspScan(ipRange:RTSP):Observable<[RTSP]> {
    return this.http.post<[RTSP]>(environment.LocalServerURL + "/api/v1/rtspscan", ipRange);
  }
}
