import { Component } from '@angular/core';
import { AdminUnitsTable } from '../../components/units_table/units_table';
import { AdminChessboard } from '../../components/chessboard/chessboard';
import { AdminStats } from '../../components/stats/stats';

@Component({
  selector: 'app-admin',
  standalone: true,
  imports: [AdminUnitsTable, AdminChessboard, AdminStats],
  templateUrl: './dashboard.html',
})
export class AdminDashboard { }
