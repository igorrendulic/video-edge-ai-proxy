import { HttpInterceptor, HttpRequest, HttpHandler, HttpEvent, HttpResponse, HttpErrorResponse } from '@angular/common/http';
import { Observable, throwError } from 'rxjs';
import { map, catchError } from 'rxjs/operators';
import { CookieService } from 'ngx-cookie-service';
import { Injectable } from '@angular/core';
import { LoaderService } from '../components/loader/loader.component';
import { Router } from '@angular/router';

@Injectable({ providedIn: 'root'})
export class ChrysHttpInterceptor implements HttpInterceptor {

    constructor(private cookieService:CookieService, public loaderService:LoaderService, private router:Router) {}

    intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
        
        if (!req.headers.has("x-skip-interceptor")) {
            this.loaderService.show();
        }

        let token = this.cookieService.get("chrys-token");

        if (token) {
            req = req.clone({ headers: req.headers.set('Authorization', token) });
        }
        if (!req.headers.has('Content-Type')) {
            req = req.clone({ headers: req.headers.set('Content-Type', 'application/json') });
        }
        req = req.clone({ headers: req.headers.set('Accept', 'application/json') });

        return next.handle(req).pipe(
            map((event: HttpEvent<any>) => {
                if (event instanceof HttpResponse) {
                    // console.log('event--->>>', event);
                    this.loaderService.hide();
                }
                return event;
            }),
            catchError((error: HttpErrorResponse) => {
                this.loaderService.hide();

                let data = {};
                data = {
                    message: error && error.error && error.error.message ? error.error.message : '',
                    status: error && error.error && error.error.code ? error.error.code : '0',
                };
                if (error.status == 401) {
                    this.router.navigate(['/user/login']);
                    return throwError(data);
                }

                if (error.status == 0) { // server unreachable
                    
                }
                
                return throwError(data);
            })
        );
    }
    
}