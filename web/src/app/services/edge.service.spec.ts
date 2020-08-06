import { TestBed } from '@angular/core/testing';

import { EdgeService } from './edge.service';

describe('EdgeService', () => {
  let service: EdgeService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(EdgeService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
