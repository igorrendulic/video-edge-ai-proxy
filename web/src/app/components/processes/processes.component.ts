import { Component, OnInit } from '@angular/core';
import { EdgeService } from 'src/app/services/edge.service';
import { StreamProcess } from 'src/app/models/StreamProcess';

@Component({
  selector: 'app-processes',
  templateUrl: './processes.component.html',
  styleUrls: ['./processes.component.scss']
})
export class ProcessesComponent implements OnInit {

  processes: [StreamProcess];
  showProcesses:Boolean = false;

  constructor(private edgeService:EdgeService) {}

  ngOnInit(): void {
    this.loadProcesses();
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
