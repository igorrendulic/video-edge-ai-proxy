import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';
import { DashboardComponent } from './components/dashboard/dashboard.component';
import { AuthGuard } from './interceptors/auth.interceptor';
import { ProcessesComponent } from './components/processes/processes.component';
import { ProcessDetailsComponent } from './components/process-details/process-details.component';
import { ProcessAddComponent } from './components/process-add/process-add.component';


const routes: Routes = [
  {path: '', redirectTo: '/local/processes', pathMatch: 'full'},
  {path: 'local', component: DashboardComponent, 
    children: [
      { path: "processes", component: ProcessesComponent},
      { path: "process/:name", component: ProcessDetailsComponent},
      { path: "addrtsp", component: ProcessAddComponent},
    ]
  }
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule { }
