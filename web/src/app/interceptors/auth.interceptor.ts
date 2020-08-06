import { Injectable } from "@angular/core";
import { CanActivate, ActivatedRouteSnapshot, RouterStateSnapshot, Router } from '@angular/router';
import { CookieService } from 'ngx-cookie-service';

@Injectable({ providedIn: 'root'})
export class AuthGuard implements CanActivate {

    constructor(private cookie:CookieService, private router:Router) {}

    canActivate(route: ActivatedRouteSnapshot, state: RouterStateSnapshot) {
        let ourcookie = this.cookie.get("chrys-token");
        
        if (ourcookie) {
            return true;
        }

        this.router.navigate(['/user/login']);
        return false;
    }

}