import { Component, OnInit } from '@angular/core';
import { FormArray, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { MatDialog } from '@angular/material/dialog';
import { Router } from '@angular/router';
import { NotificationsService } from 'angular2-notifications';
import { AppProcess, PortMap } from 'src/app/models/AppProcess';
import { EdgeService } from 'src/app/services/edge.service';
import { WaitDialogComponent } from '../shared/wait-dialog/wait-dialog.component';

@Component({
  selector: 'app-app-add',
  templateUrl: './app-add.component.html',
  styleUrls: ['./app-add.component.scss']
})
export class AppAddComponent implements OnInit {

  appForm:FormGroup;
  // dockerTags:DockerTag[] = [
  //   {value: "", viewValue:"default"}
  // ]
  runtimeSelected:string = '';
  submitted:Boolean = false;
  errorMessage:string;
  loadingMessage:string;

  constructor(private _formBuilder:FormBuilder, 
    private edgeService:EdgeService, 
    private router:Router,
    private notifService:NotificationsService,
    public dialog:MatDialog) { 

    this.appForm = this._formBuilder.group({
      name: [null, [Validators.required, Validators.minLength(3)]],
      docker_user: [null],
      docker_repository: [null, Validators.required],
      docker_version: [null, Validators.required],
      env_vars: this._formBuilder.array([]),
      mount: this._formBuilder.array([]),
      port_mappings: this._formBuilder.array([]),
      runtime: [null],
    });
  }

  mounts(): FormArray {
    return this.appForm.get("mount") as FormArray;
  }

  newMount(): FormGroup {
    return this._formBuilder.group({
      name: [null, Validators.required],
      value: [null, Validators.required],
    })
  }

  addMount() {
    this.mounts().push(this.newMount());
  }

  removeMount(i:number) {
    this.mounts().removeAt(i);
  }

  portMaps(): FormArray {
    return this.appForm.get("port_mappings") as FormArray
  }

  newPortMap(): FormGroup {
    return this._formBuilder.group({
      exposed: [null, Validators.required],
      map_to: [null, Validators.required],
    })
  }

  addPortMap() {
    this.portMaps().push(this.newPortMap());
  }
   
  removePortMap(i:number) {
    this.portMaps().removeAt(i);
  }


  envVars() : FormArray {
    return this.appForm.get("env_vars") as FormArray
  }

  newEnvVar(): FormGroup {
    return this._formBuilder.group({
      name: [null, Validators.required],
      value: [null, Validators.required],
    })
  }

  addEnvVar() {
    this.envVars().push(this.newEnvVar());
  }
   
  removeEnvVar(i:number) {
    this.envVars().removeAt(i);
  }

  get f() { return this.appForm.controls; }

  ngOnInit(): void {
  }

  downloadApp(app:AppProcess, tag:string,version:string, title:string, message:string) {
    const dialogReg = this.dialog.open(WaitDialogComponent, {
      maxWidth: "400px",
      disableClose: true,
      data: {
        title: title,
        message: message
      }
    });

    console.log("inspect app: ", app);

    this.edgeService.pullDockerImage(tag, version).subscribe(pullData => {
      this.loadingMessage = pullData.response;
      // popup  window with Next button
      console.log("pulled successfully: ", pullData);
      this.startApp(app);
      dialogReg.close();
    }, pullErr => {
      dialogReg.close();
      console.error(pullErr);
      this.loadingMessage = pullErr
      
      this.notifService.error("Please execute this command in your terminal: docker pull " + tag +  ":" + version );
    });

  }

  startApp(app:AppProcess) {
    const dialogReg = this.dialog.open(WaitDialogComponent, {
      maxWidth: "400px",
      disableClose: true,
      data: {
        title: "Install",
        message: "Starting the app"
      }
    });

    this.edgeService.installApp(app).subscribe(resp => {
      
      dialogReg.close();
      this.router.navigate(['/local/processes'],  { queryParams: {tab: 1}});

    }, error => {
      console.log(error);
      dialogReg.close();
      this.loadingMessage = error.message;
      this.notifService.error("Start failed");
    })
  }

  onSubmit() {
    this.submitted = true;
    if (!this.appForm.valid) {
      return
    }

    let app:AppProcess = this.appForm.value;

    let imageTag = app.docker_repository
    let imageVersion = app.docker_version
    if (app.docker_user) {
      imageTag = app.docker_user + "/" + imageTag
    }

    // // convert to ints from form strings
    if (app.port_mappings) {
      let portMappings:PortMap[] = [];
      app.port_mappings.forEach(pm => {
        let portMap:PortMap = {
          exposed: Number(pm.exposed),
          map_to: Number(pm.map_to),
        }
        portMappings.push(portMap);
      });
      app.port_mappings = portMappings;
    }
    
    this.downloadApp(app,imageTag, imageVersion, "Downloading app", "Do not close this browser window. This may take a while. Please wait...")

  }

}
