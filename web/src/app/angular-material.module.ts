import { NgModule, CUSTOM_ELEMENTS_SCHEMA } from '@angular/core';
import { CommonModule } from '@angular/common';

import { MatButtonModule } from '@angular/material/button';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatIconModule } from '@angular/material/icon';
import { MatBadgeModule } from '@angular/material/badge';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatListModule } from '@angular/material/list';
import { MatGridListModule } from '@angular/material/grid-list';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatRadioModule } from '@angular/material/radio';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatNativeDateModule } from '@angular/material/core';
import { MatChipsModule } from '@angular/material/chips';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatTableModule } from '@angular/material/table';
import { MatPaginatorModule } from '@angular/material/paginator';
import { MatCardModule } from '@angular/material/card';
import { MatTreeModule } from '@angular/material/tree';
import { MatMenuModule } from '@angular/material/menu';
import { MatTabsModule } from '@angular/material/tabs';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import {MatStepperModule} from '@angular/material/stepper';
import {MatExpansionModule} from '@angular/material/expansion';
import {MatProgressSpinnerModule} from '@angular/material/progress-spinner';
import {MatCheckboxModule} from '@angular/material/checkbox';


@NgModule({
    imports: [
       CommonModule,
       MatButtonModule,
       MatToolbarModule,
       MatIconModule,
       MatSidenavModule,
       MatBadgeModule,
       MatListModule,
       MatGridListModule,
       MatFormFieldModule,
       MatInputModule,
       MatSelectModule,
       MatRadioModule,
       MatDatepickerModule,
       MatNativeDateModule,
       MatChipsModule,
       MatTooltipModule,
       MatTableModule,
       MatPaginatorModule,
       MatCardModule,
       MatTreeModule,
       MatStepperModule,
       MatMenuModule,
       MatTabsModule,
       MatProgressBarModule,
       MatProgressSpinnerModule,
       MatExpansionModule,
       MatCheckboxModule
    ],
    exports: [
       MatButtonModule,
       MatToolbarModule,
       MatIconModule,
       MatSidenavModule,
       MatBadgeModule,
       MatListModule,
       MatGridListModule,
       MatInputModule,
       MatFormFieldModule,
       MatSelectModule,
       MatRadioModule,
       MatDatepickerModule,
       MatChipsModule,
       MatTooltipModule,
       MatTableModule,
       MatPaginatorModule,
       MatCardModule,
       MatTreeModule,
       MatMenuModule,
       MatTabsModule,
       MatProgressBarModule,
       MatStepperModule,
       MatProgressSpinnerModule,
       MatExpansionModule,
       MatCheckboxModule
    ],
    providers: [
       MatDatepickerModule,
    ]
 })
 
 export class AngularMaterialModule { }