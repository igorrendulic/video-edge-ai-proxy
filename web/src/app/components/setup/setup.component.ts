import { MediaMatcher } from '@angular/cdk/layout';
import { ThrowStmt } from '@angular/compiler';
import { toBase64String } from '@angular/compiler/src/output/source_map';
import { ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { Router } from '@angular/router';
import { NotificationsService } from 'angular2-notifications';
import { Observable, Subject } from 'rxjs';
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
  title:string = ""

  mobileQuery: MediaQueryList;

  imageUpgrade = new Subject<ImageUpgrade>();
  imageUpgrade$ = this.imageUpgrade.asObservable();

  imageUpgrades:ImageUpgrade[]= [];

  expectedResponses:number;
  gotResponses:number = 0;

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

    const dialogRef = this.dialog.open(WaitDialogComponent, {
      maxWidth: "400px",
      disableClose: true,
      data: {
          title: "Please wait...",
          message: "Checking for updates"}
      });

    this.initialSetup();

    this.imageUpgrade$.subscribe(data => {
      this.gotResponses += 1;
      this.imageUpgrades.push(data);
      if (this.gotResponses == this.expectedResponses) {
        // close dialog, choose which option to proceed with
        dialogRef.close();

        if (!data.has_upgrade && data.has_image) {

          // no upgrades, latest version available
          this.router.navigate(['/local/processes']);
        } else if (!data.has_upgrade && !data.has_image) {

          // no images found...initial setup
          this.title = "Initial setup. Please choose a camera type to install";
        } else if (data.has_image && data.has_upgrade) {

          // upgrade available
          this.title = "Upgrade available";
          const dialogRef = this.dialog.open(ConfirmDialogComponent, {
            maxWidth: "400px",
            data: {
                title: "New version for RTSP cameras available " + data.highest_remote_version,
                message: "Would you like to download newer version of RTSP camera images? Updates may contain some upgrades and/or bug fixes. Your camera operations are not affected by this process. It's just a download."}
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
        }
      }
    });
  }

  clickRTSP() {
    let rtspImage = GlobalVars.CameraTypes.get("rtsp");
    console.log("rtspImage from global vars: ", rtspImage);
    
    let found = false;
    this.imageUpgrades.forEach(upgrade => {
        if (upgrade.name == rtspImage) {
          found = true;
          this.pullImage(upgrade, "Downloading RTSP camera container","Please wait. This may take a few minutes.");
        }
    });
    
    if (!found) {
      console.error("failed to find a docker image when doing continer upgrade", rtspImage, GlobalVars.CameraTypes);
      this.notifService.error("Failed to find the right container image. Try hard refreshing.");
    }
  }

  initialSetup() {
    this.expectedResponses = GlobalVars.CameraTypes.size;

    GlobalVars.CameraTypes.forEach((value,key) => {
      console.log(key,value);
      this.edgeService.getDeviceDockerImages(value).subscribe(data => {
        this.imageUpgrade.next(data);
      });
    });
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

    this.edgeService.pullDockerImage(data.name, data.highest_remote_version).subscribe(pullData => {
      this.loadingMessage = pullData.response;
      // popup  window with Next button
      this.router.navigate(['/local/processes']);     
      dialogReg.close();
    }, pullErr => {
      dialogReg.close();
      console.error(pullErr);
      this.loadingMessage = pullErr
      this.loading = false;
      this.notifService.error("Please execute this command in your terminal: docker pull " + data.name +  ":" + data.highest_remote_version );
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
