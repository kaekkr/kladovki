import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { ClientHeader } from './layout/header/header';
import { ClientFooter } from './layout/footer/footer';
import { AccountStatus } from './layout/account-status/account-status';
import { ForcePasswordChange } from '../../shared/components/change-password/force-password-change';

@Component({
  selector: 'app-client',
  standalone: true,
  imports: [RouterOutlet, ClientHeader, ClientFooter, AccountStatus, ForcePasswordChange],
  templateUrl: './client.html',
})
export class Client { }
