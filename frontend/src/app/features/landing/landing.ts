import { Component } from '@angular/core';
import { Header } from './header/header';
import { Hero } from './hero/hero';
import { Solutions } from './solutions/solutions';
import { Features } from './features/features';
import { Contact } from './contact/contact';
import { Partners } from './partners/partners';
import { Footer } from './footer/footer';

@Component({
  selector: 'app-landing',
  imports: [Header, Hero, Solutions, Features, Contact, Partners, Footer],
  templateUrl: './landing.html',
})
export class Landing { }
