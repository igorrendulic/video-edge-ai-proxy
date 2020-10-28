import { Component, OnInit } from '@angular/core';
import { EdgeService } from 'src/app/services/edge.service';
import { StreamProcess } from 'src/app/models/StreamProcess';
import { GlobalVars } from 'src/app/models/RTSP';
import { Subject } from 'rxjs';
import { ImageUpgrade } from 'src/app/models/ImageUpgrade';
import { semver } from 'semver';
import { ThrowStmt } from '@angular/compiler';
import { SelectionModel } from '@angular/cdk/collections';
import { NotificationsService } from 'angular2-notifications';

@Component({
  selector: 'app-processes',
  templateUrl: './processes.component.html',
  styleUrls: ['./processes.component.scss']
})
export class ProcessesComponent implements OnInit {

  processes: StreamProcess[] = [];
  showProcesses:Boolean = false;

  selection = new SelectionModel<StreamProcess>(true, []);

  upgrading:[] = [];
  disabledUpgradeButton:boolean = false;

  constructor(private edgeService:EdgeService, private notifService:NotificationsService) {}

  ngOnInit(): void {
    this.loadProcesses();
    this.getUpgradableProcesses();
  }

  getUpgradableProcesses() {
    this.edgeService.getRTSPProcessUpgrades().subscribe(data => {
      console.log(data);
      if (data.length > 0) {
        this.processes = data;
      }
    });
  }

  // checks if all upgradable processes selected
  isAllUpgradesSelected() {
    const numSelected = this.selection.selected.length;
    var numUpgrades = 0;
    this.processes.forEach(process => {
      if (process.upgrade_available) {
        numUpgrades++;
      }
    });
    return numSelected === numUpgrades;
  }

   /** Selects all upgradable processes if they are not all selected; otherwise clear selection. */
   selectToggle() {
    this.isAllUpgradesSelected() ?
        this.selection.clear() :
        this.processes.forEach(row => { if (row.upgrade_available) {this.selection.select(row)} });
  }

  upgrade() {
    this.disabledUpgradeButton = true;
    let allUpgradesNumber = this.selection.selected.length;
    let upgradedSoFar = 0;
    if (this.selection.selected.length > 0) {
      this.selection.selected.forEach(process => {

        // set all to upgrading status
        this.processes.forEach(listedProcess => {
          if (listedProcess.name == process.name) {
            listedProcess.upgrading_now = true;
          }
        });

        // start upgrde process for each selected process
        this.edgeService.upgradeProcessContainer(process).subscribe(data => {
          console.log("upgrade success: ", data, process);

          this.processes.forEach(listedProcess => {
            if (listedProcess.name == data.name) {
              listedProcess.upgrading_now = false;
            }
          });

          upgradedSoFar += 1;
          console.log("upgraded so far", upgradedSoFar, allUpgradesNumber);
          if (allUpgradesNumber == upgradedSoFar) {
            this.selection.clear();
            this.getUpgradableProcesses();
            this.disabledUpgradeButton = false;
          }
        
        // set to up
          
        }, error => {
          upgradedSoFar += 1;
          console.error("failed to upgrade container", error);
          this.notifService.error("Error", "Failed to upgrade container: " + error.message, {
            clickToClose: true,
            clickIconToClose: true
          });
          if (allUpgradesNumber == upgradedSoFar) {
            this.selection.clear();
            this.getUpgradableProcesses();
            this.disabledUpgradeButton = false;
          }
        })
      });
    }
  }

  loadProcesses() {
    this.edgeService.listRTSP().subscribe(list => {
      this.processes = list;
      if (list.length > 0) {
        this.showProcesses = true;
      }
    }, error => {
      this.showProcesses = false;
      console.error(error);
    })
  }

}
