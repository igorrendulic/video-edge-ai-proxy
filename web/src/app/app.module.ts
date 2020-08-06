import { BrowserModule } from '@angular/platform-browser';
import { NgModule, CUSTOM_ELEMENTS_SCHEMA } from '@angular/core';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { environment } from 'src/environments/environment';
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

const bullets = [{
  Title: 'Quick and free sign-up',
  Subtitle: 'Sing up for a 30 day trial',
},{
  Title: 'Faster development',
  Subtitle:'Focus on core business instead of maintaining pipelines',
},{
  Title: 'End-to-End Solution',
  Subtitle: 'Collect, analyze and process streaming video',
},{
  Title: 'Save up to 70%',
  Subtitle: "Video streaming doesn't have to be so expensive"
},{
  Title: 'Fully secure',
  Subtitle: 'Peace of mind when streaming from home'
}];

@NgModule({
  declarations: [
    AppComponent,
    LoaderComponent,
    DashboardComponent,
    ProcessesComponent,
    ProcessDetailsComponent,
    ProcessAddComponent,
    ConfirmDialogComponent,
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
