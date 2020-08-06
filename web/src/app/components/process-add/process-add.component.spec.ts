import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { ProcessAddComponent } from './process-add.component';

describe('ProcessAddComponent', () => {
  let component: ProcessAddComponent;
  let fixture: ComponentFixture<ProcessAddComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ ProcessAddComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ProcessAddComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
