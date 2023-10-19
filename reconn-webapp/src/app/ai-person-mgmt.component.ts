import { HttpErrorResponse } from '@angular/common/http';
import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { EMPTY, Observable, ReplaySubject, fromEvent, of, shareReplay } from 'rxjs';
import { catchError, exhaustMap, map, startWith, tap } from 'rxjs/operators';
import { AIPerson, ChatService, ReadbackResponse, ReadbackService, UpdateAIPersonContextPromptByIDParams, VoiceModelResponse } from './chat.module';

@Component({
  selector: 'ai-person-mgmt',
  templateUrl: './ai-person-mgmt.component.html',
})
export class AIPersonManagementComponent implements OnInit {
  // Create a new AI person.
  createUserID = '';
  createAIPersonName = 'Balaam';
  createContextPrompt = 'You are Balaam son of Beor, you are an official appointed by king Balak.'

  // AI person list.
  @ViewChild('refreshButton', { static: true }) refreshButton!: ElementRef;
  listUserID = '';
  listResp: Observable<AIPerson[]> = EMPTY;

  // Update AI person.
  updateAIPersonID = '';
  updateAIPersonContextPrompt = '';

  constructor(private chatService: ChatService) {
  }

  ngOnInit() {
    this.listResp = fromEvent(this.refreshButton.nativeElement, 'click')
      .pipe(
        startWith(undefined),
        exhaustMap((click) => this.chatService.listAIPersons(Number(this.listUserID))),
        shareReplay({ bufferSize: 1, refCount: true })
      );
  }

  createAIPersonClick() {
    this.chatService.createAIPerson({ UserID: Number(this.createUserID), Name: this.createAIPersonName, ContextPrompt: this.createContextPrompt } as AIPerson).pipe(
      catchError((error) => {
        alert(JSON.stringify(error));
        return EMPTY;
      })
    ).subscribe((result) => {
      alert(JSON.stringify(result));
    });
  }

  updateAIPersonClick() {
    this.chatService.updateAIPerson(Number(this.updateAIPersonID), { ContextPrompt: this.updateAIPersonContextPrompt } as UpdateAIPersonContextPromptByIDParams).pipe(
      catchError((error) => {
        alert(JSON.stringify(error));
        return EMPTY;
      })
    ).subscribe((result) => {
      alert(JSON.stringify(result));
    });
  }
}
