import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { AdminHeader } from './layout/header/header';
import { AdminSidebar } from './layout/sidebar/sidebar';
import { AdminFooter } from './layout/footer/footer';
import { OsiRulesPrompt } from './components/osi-rules-prompt/osi-rules-prompt';
import { ForcePasswordChange } from '../../shared/components/change-password/force-password-change';

@Component({
  selector: 'app-admin',
  standalone: true,
  imports: [RouterOutlet, AdminHeader, AdminSidebar, AdminFooter, OsiRulesPrompt, ForcePasswordChange],
  templateUrl: './admin.html',
})
export class Admin { }
