import { MediaMatcher } from '@angular/cdk/layout';
import { toBase64String } from '@angular/compiler/src/output/source_map';
import { ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { Router } from '@angular/router';
import { NotificationsService } from 'angular2-notifications';
import { ImageUpgrade } from 'src/app/models/ImageUpgrade';
import { GlobalVars } from 'src/app/models/RTSP';
import { EdgeService } from 'src/app/services/edge.service';
import { ConfirmDialogComponent } from '../shared/confirm-dialog/confirm-dialog.component';
import { WaitDialogComponent } from '../shared/wait-dialog/wait-dialog.component';

const dockerTags = ["chryscloud/chrysedgeproxy"]

@Component({
  selector: 'app-setup',
  templateUrl: './setup.component.html',
  styleUrls: ['./setup.component.scss']
})
export class SetupComponent implements OnInit, OnDestroy {

  loading:boolean = false;
  loadingMessage:string = "Please wait ... checking settings";

  mobileQuery: MediaQueryList;

  private _mobileQueryListener: () => void;

  constructor(changeDetectorRef: ChangeDetectorRef, 
      media: MediaMatcher,
      private router:Router, 
      private edgeService:EdgeService,  
      private notifService:NotificationsService,
      public dialog:MatDialog) { 

    this.mobileQuery = media.matchMedia('(max-width: 600px)');
    this._mobileQueryListener = () => changeDetectorRef.detectChanges();
    this.mobileQuery.addListener(this._mobileQueryListener);

  }

  ngOnInit(): void {
    this.loading = true;
  }

  initialSetup(cameraType:string) {
    if (GlobalVars.CameraTypeRTSP.has(cameraType)) {
      console.log("rtsp fond");
    } else {
      console.log("not found");
    }
  }

  checkSettings(cameraType:string) {
     this.edgeService.getDockerImages("chryscloud/chrysedgeproxy").subscribe(data => {
       
       if (!data.has_image) {
        this.pullImage(data, "Please wait...Inital setup in progress","Do not close this window until setup finishes.");
       } else {
         // has image, check if upgrade available
         if (data.has_upgrade) {
           this.loadingMessage = "Please wait ... upgrading image";

          const dialogRef = this.dialog.open(ConfirmDialogComponent, {
            maxWidth: "400px",
            data: {
                title: "New version of docker image for RTSP cameras available",
                message: "Would you like to download newer versions of RTSP camera images? Updates may contain some upgrades and/or bug fixes. Your camera operations are not affected by this process."}
            });
        
            // upgrade yes/no
            dialogRef.afterClosed().subscribe(dialogResult => {
              // if user pressed yes dialogResult will be true, 
              // if he pressed no - it will be false
              if (dialogResult) {
                this.pullImage(data, "Please wait ...", "Do not close this window until upgrade finishes.");
              } else {
                // no upgrading - proceed to camera list view
                this.router.navigate(['/local/processes']);      
              }
            });

         } else {
           // all ok, move along
          this.loading = false;
          this.router.navigate(['/local/processes']);
         }
       }
     }, error => {
       console.error(error);
       this.loadingMessage = error
      this.loading = false;
     })
  }

  ngOnDestroy(): void {
    this.mobileQuery.removeListener(this._mobileQueryListener);
  }

  pullImage(data:ImageUpgrade, title:string, message:string) {
    const dialogReg = this.dialog.open(WaitDialogComponent, {
      maxWidth: "400px",
      data: {
        title: title,
        message: message
      }
    });

    this.edgeService.pullDockerImage("chryscloud/chrysedgeproxy", data.highest_remote_version).subscribe(pullData => {
      this.loadingMessage = pullData.response;
      // popup  window with Next button
      this.router.navigate(['/local/processes']);     
      dialogReg.close();
    }, pullErr => {
      dialogReg.close();
      console.error(pullErr);
      this.loadingMessage = pullErr
      this.loading = false;
      this.openDialog("Initial setup failed", "Please execute this command in your terminal: docker pull chryscloud/chrysedgeproxy" + data.highest_remote_version + ".");
    });

  }

  openDialog(title:string, message:string) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
    maxWidth: "400px",
    data: {
        title: title,
        message: message}
    });

    // listen to response
    dialogRef.afterClosed().subscribe(dialogResult => {
      // if user pressed yes dialogResult will be true, 
      // if he pressed no - it will be false
      if (dialogResult) {
        this.loading = false;
        this.router.navigate(['/local/processes']);
      }
    });
  }

}
