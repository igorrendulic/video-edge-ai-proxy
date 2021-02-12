import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { AppAddComponent } from './app-add.component';

describe('AppAddComponent', () => {
  let component: AppAddComponent;
  let fixture: ComponentFixture<AppAddComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ AppAddComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(AppAddComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
