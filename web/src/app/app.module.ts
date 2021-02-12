import { BrowserModule } from '@angular/platform-browser';
import { NgModule, CUSTOM_ELEMENTS_SCHEMA } from '@angular/core';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { FlexLayoutModule } from '@angular/flex-layout';
import { CookieService } from 'ngx-cookie-service';
import { HttpClientModule, HTTP_INTERCEPTORS } from '@angular/common/http';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { AngularMaterialModule } from './angular-material.module';
import { LoaderComponent, LoaderService } from './components/loader/loader.component';
import { DashboardComponent } from './components/dashboard/dashboard.component';
import { ChrysHttpInterceptor } from './interceptors/http.interceptor';
import { ProcessesComponent } from './components/processes/processes.component';
import { ProcessDetailsComponent } from './components/process-details/process-details.component';
import { ProcessAddComponent } from './components/process-add/process-add.component';
import { ConfirmDialogComponent } from './components/shared/confirm-dialog/confirm-dialog.component';
import { SettingsComponent } from './components/settings/settings.component';
import { SetupComponent } from './components/setup/setup.component';
import { NotifyDialogComponent } from './components/shared/notify-dialog/notify-dialog.component';
import { WaitDialogComponent } from './components/shared/wait-dialog/wait-dialog.component';
import { SimpleNotificationsModule } from 'angular2-notifications';
import { AppAddComponent } from './components/app-add/app-add.component';
import { AppDetailsComponent } from './components/app-details/app-details.component';

@NgModule({
  declarations: [
    AppComponent,
    LoaderComponent,
    DashboardComponent,
    ProcessesComponent,
    ProcessDetailsComponent,
    ProcessAddComponent,
    ConfirmDialogComponent,
    SettingsComponent,
    SetupComponent,
    NotifyDialogComponent,
    WaitDialogComponent,
    AppAddComponent,
    AppDetailsComponent,
  ],
  imports: [
    BrowserModule,
    AngularMaterialModule,
    AppRoutingModule,
    FlexLayoutModule,
    HttpClientModule,
    FormsModule,
    ReactiveFormsModule,
    BrowserAnimationsModule,
    SimpleNotificationsModule.forRoot(),
  ],
  entryComponents: [
    ConfirmDialogComponent
  ],
  providers: [
    CookieService,
    LoaderService,
    {provide: HTTP_INTERCEPTORS, useClass: ChrysHttpInterceptor, multi: true}
  ],
  bootstrap: [AppComponent],
  schemas: [CUSTOM_ELEMENTS_SCHEMA]
})
export class AppModule { }
