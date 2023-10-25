import { HttpErrorResponse } from '@angular/common/http';
import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { EMPTY, Observable, ReplaySubject, fromEvent, of, shareReplay } from 'rxjs';
import { catchError, exhaustMap, map, startWith, tap } from 'rxjs/operators';
import { AudioRecorderService, ErrorCase, OutputFormat } from './audio_recorder.module';
import { AIPerson, ChatService, ListConversationsRow, ReadbackResponse, ReadbackService, UpdateAIPersonContextPromptByIDParams, VoiceModelResponse } from './chat.module';

@Component({
  selector: 'conversation-mgmt',
  templateUrl: './conversation-mgmt.component.html',
})
export class ConversationManagementComponent implements OnInit {
  aiPersonID = '';
  // Post text message.
  textMessage = '';
  // Post voice message.
  recordingInProgress = false;
  recordButtonCaption = 'Start recording';
  recording?: Blob;
  recordingStatus = new ReplaySubject<string>(1);
  // Conversation history.
  @ViewChild('listConversationButton', { static: true }) listConversationButton!: ElementRef;
  listConversationAIPersonID = '';
  listResp: Observable<ListConversationsRow[]> = EMPTY;
  outputFileBlob = new ReplaySubject<Blob>(1);
  outputFileBlobSrc: Observable<string>;

  constructor(readonly recorderService: AudioRecorderService, readonly chatService: ChatService) {
    recorderService.recorderError.subscribe((error) => {
      this.recordingStatus.next(JSON.stringify(error));
    });
    this.outputFileBlobSrc = this.outputFileBlob.pipe(
      map((input: Blob) => this.blobToUrl(input)),
    );
  }

  recordButtonClick() {
    if (this.recordingInProgress) {
      this.recorderService.stopRecording(OutputFormat.BLOB).then((blob) => {
        console.log('recording blob', blob);
        this.recording = blob as Blob;
        this.recordingStatus.next("Recording size: " + this.recording.size / 1024 + "KB");
      });
      this.recordingInProgress = false;
      this.recordButtonCaption = 'Start over';
    } else {
      this.recorderService.startRecording();
      this.recordingInProgress = true;
      this.recordButtonCaption = 'Finish recording';
    }
  }

  ngOnInit() {
    this.listResp = fromEvent(this.listConversationButton.nativeElement, 'click')
      .pipe(
        startWith(undefined),
        exhaustMap((click) => this.chatService.listConversation(Number(this.aiPersonID), 200)),
        shareReplay({ bufferSize: 1, refCount: true })
      );
  }

  sendTextClick() {
    this.chatService.postTextMessage(Number(this.aiPersonID), this.textMessage).pipe(
      map((resp) => resp),
      catchError((err) => of(err))
    ).subscribe((result: unknown) => {
      alert(JSON.stringify(result));
    });
  }

  sendVoiceNoteClick() {
    if (!this.recording || !this.aiPersonID) {
      return;
    }
    console.log('input recording', this.recording);
    this.chatService.postVoiceMessage(Number(this.aiPersonID), this.recording).pipe(
      map((resp) => resp),
      catchError((err) => of(err))
    ).subscribe((result: unknown) => {
      alert(JSON.stringify(result));
    });
  }

  getVoiceFile(outputFileName?: string) {
    if (!outputFileName) {
      return;
    }
    this.chatService.getVoiceOutputFile(outputFileName)
      .pipe(
        catchError((err: HttpErrorResponse) => {
          alert('Error: ' + JSON.stringify(err));
          return EMPTY;
        })
      )
      .subscribe((val) => {
        this.outputFileBlob.next(val);
      });
  }

  blobToUrl(input: Blob) {
    const blob = new Blob([input], { type: "audio/wav" });
    const blobUrl = URL.createObjectURL(blob);
    return blobUrl;
  }
}
