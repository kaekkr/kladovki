import { Component, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { AuthService } from '../../../../core/auth/auth.service';
import { DocumentService } from '../../../../core/services/document';
import { AdminDocument, DocumentType } from '../../../../core/models/document';

@Component({
  selector: 'app-admin-documents',
  standalone: true,
  imports: [DatePipe],
  templateUrl: './documents.html',
})
export class AdminDocuments {
  private authService = inject(AuthService);
  private documentService = inject(DocumentService);

  documents = signal<AdminDocument[]>([]);
  loading = signal(true);
  error = signal(false);
  search = signal('');
  typeFilter = signal<DocumentType | 'all'>('all');

  filteredDocuments = computed(() => {
    const q = this.search().trim().toLowerCase();
    const type = this.typeFilter();
    return this.documents().filter((d) => {
      const matchesSearch =
        !q ||
        d.storage_number.toLowerCase().includes(q) ||
        d.user_name.toLowerCase().includes(q) ||
        d.user_phone.toLowerCase().includes(q) ||
        d.title.toLowerCase().includes(q);
      const matchesType = type === 'all' || d.type === type;
      return matchesSearch && matchesType;
    });
  });

  counts = computed(() => {
    const list = this.documents();
    return {
      total: list.length,
      contract: list.filter((d) => d.type === 'contract').length,
      receipt: list.filter((d) => d.type === 'receipt').length,
      act: list.filter((d) => d.type === 'act').length,
    };
  });

  constructor() {
    this.load();
  }

  private load(): void {
    const jkId = this.authService.currentUser()?.jk_id;
    if (!jkId) {
      this.loading.set(false);
      this.error.set(true);
      return;
    }
    this.loading.set(true);
    this.error.set(false);
    this.documentService.listByJK(jkId).subscribe({
      next: (items) => {
        this.documents.set(items);
        this.loading.set(false);
      },
      error: () => {
        this.error.set(true);
        this.loading.set(false);
      },
    });
  }

  setSearch(value: string): void {
    this.search.set(value);
  }

  setTypeFilter(type: DocumentType | 'all'): void {
    this.typeFilter.set(type);
  }

  typeLabel(type: DocumentType): string {
    switch (type) {
      case 'contract':
        return 'Договор';
      case 'receipt':
        return 'Квитанция';
      case 'act':
        return 'Акт';
      default:
        return type;
    }
  }

  typeIcon(type: DocumentType): string {
    switch (type) {
      case 'contract':
        return 'fa-file-contract';
      case 'receipt':
        return 'fa-receipt';
      case 'act':
        return 'fa-file-signature';
      default:
        return 'fa-file';
    }
  }

  formatMoney(value: number): string {
    return new Intl.NumberFormat('ru-RU').format(value) + ' ₸';
  }

  /** Stub — later open PDF / download */
  openDocument(doc: AdminDocument): void {
    // For now just log; later: window.open(...) or generate PDF
    console.log('Open document', doc);
    alert(`Документ: ${doc.title}\n(генерация PDF будет позже)`);
  }
}
