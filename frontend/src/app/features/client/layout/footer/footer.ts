import { Component } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';

interface NavItem {
  link: string;
  label: string;
  icon: string;
  exact: boolean;
}

@Component({
  selector: 'app-client-footer',
  standalone: true,
  imports: [RouterLink, RouterLinkActive],
  templateUrl: './footer.html',
})
export class ClientFooter {
  readonly items: readonly NavItem[] = [
    { link: '/app', label: 'Главная', icon: 'fa-house', exact: true },
    { link: '/app/storages', label: 'Кладовки', icon: 'fa-th', exact: false },
    { link: '/app/rentals', label: 'Аренды', icon: 'fa-file-contract', exact: false },
    { link: '/app/payments', label: 'Платежи', icon: 'fa-wallet', exact: false },
  ];
}
