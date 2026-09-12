import { Component, inject } from '@angular/core';
import { AuthService } from '../../../../core/auth/auth.service';

@Component({
  selector: 'app-client-header',
  standalone: true,
  templateUrl: './header.html',
})
export class ClientHeader {
  private auth = inject(AuthService);

  name = this.auth.currentUser()?.full_name || 'Житель';
}
