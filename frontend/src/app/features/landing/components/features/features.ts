import { Component } from '@angular/core';

@Component({
  selector: 'app-features',
  standalone: true,
  templateUrl: './features.html',
})
export class Features {
  /** Decorative floor plan: true = a free storage (illustration, not live data). */
  readonly cells = [true, false, false, true, false, false, false, true, false, false, false, true];
}
