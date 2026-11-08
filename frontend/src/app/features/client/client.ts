import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { ClientHeader } from './layout/header/header';
import { ClientFooter } from './layout/footer/footer';
import { AccountStatus } from './layout/account-status/account-status';

@Component({
  selector: 'app-client',
  standalone: true,
  imports: [RouterOutlet, ClientHeader, ClientFooter, AccountStatus],
  templateUrl: './client.html',
})
export class Client { }
