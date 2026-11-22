import { Component } from '@angular/core';
import { Header } from './components/header/header';
import { Hero } from './components/hero/hero';
import { Solutions } from './components/solutions/solutions';
import { Features } from './components/features/features';
import { Contact } from './components/contact/contact';
import { Footer } from './components/footer/footer';

@Component({
  selector: 'app-landing',
  imports: [Header, Hero, Solutions, Features, Contact, Footer],
  templateUrl: './landing.html',
})
export class Landing {}
