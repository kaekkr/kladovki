import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { ClientHeader } from './layout/header/header';
import { ClientFooter } from './layout/footer/footer';

@Component({
  selector: 'app-client',
  standalone: true,
  imports: [RouterOutlet, ClientHeader, ClientFooter],
  templateUrl: './client.html',
})
export class Client { }
