package main_test

import (
	"fmt"
	"sync/atomic"

	"github.com/cloudfoundry-incubator/receptor"
	"github.com/cloudfoundry-incubator/receptor/serialization"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/tedsuo/ifrit/ginkgomon"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Desired LRP API", func() {

	BeforeEach(func() {
		receptorProcess = ginkgomon.Invoke(receptorRunner)
	})

	AfterEach(func() {
		ginkgomon.Kill(receptorProcess)
	})

	Describe("POST /desired_lrps/", func() {
		var lrpToCreate receptor.DesiredLRPCreateRequest
		var createErr error

		BeforeEach(func() {
			lrpToCreate = newValidDesiredLRPCreateRequest()
			createErr = client.CreateDesiredLRP(lrpToCreate)
		})

		It("responds without an error", func() {
			Ω(createErr).ShouldNot(HaveOccurred())
		})

		It("desires an LRP in the BBS", func() {
			Eventually(bbs.GetAllDesiredLRPs).Should(HaveLen(1))
			desiredLRPs, err := bbs.GetAllDesiredLRPs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(desiredLRPs[0].ProcessGuid).To(Equal(lrpToCreate.ProcessGuid))
		})

		It("is idempotent", func() {
			err := client.CreateDesiredLRP(lrpToCreate)
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("GET /desired_lrps/:process_guid", func() {
		var lrpRequest receptor.DesiredLRPCreateRequest
		var lrpResponse receptor.DesiredLRPResponse
		var getErr error

		BeforeEach(func() {
			lrpRequest = newValidDesiredLRPCreateRequest()
			err := client.CreateDesiredLRP(lrpRequest)
			Ω(err).ShouldNot(HaveOccurred())

			lrpResponse, getErr = client.GetDesiredLRP(lrpRequest.ProcessGuid)
		})

		It("responds without an error", func() {
			Ω(getErr).ShouldNot(HaveOccurred())
		})

		It("fetches the desired lrp with the matching process guid", func() {
			desiredLRP, err := serialization.DesiredLRPFromRequest(lrpRequest)
			Ω(err).ShouldNot(HaveOccurred())

			expectedLRPResponse := serialization.DesiredLRPToResponse(desiredLRP)
			Ω(lrpResponse).Should(Equal(expectedLRPResponse))
		})
	})

	Describe("PUT /desired_lrps/:process_guid", func() {
		var updateErr error

		instances := 6
		annotation := "update-annotation"
		routes := []string{"updated-route"}

		BeforeEach(func() {
			createLRPReq := newValidDesiredLRPCreateRequest()
			err := client.CreateDesiredLRP(createLRPReq)
			Ω(err).ShouldNot(HaveOccurred())

			update := receptor.DesiredLRPUpdateRequest{
				Instances:  &instances,
				Annotation: &annotation,
				Routes:     routes,
			}

			updateErr = client.UpdateDesiredLRP(createLRPReq.ProcessGuid, update)
		})

		It("responds without an error", func() {
			Ω(updateErr).ShouldNot(HaveOccurred())
		})

		It("updates the LRP in the BBS", func() {
			Eventually(bbs.GetAllDesiredLRPs).Should(HaveLen(1))
			desiredLRPs, err := bbs.GetAllDesiredLRPs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(desiredLRPs[0].Instances).To(Equal(instances))
			Ω(desiredLRPs[0].Routes).To(Equal(routes))
			Ω(desiredLRPs[0].Annotation).To(Equal(annotation))
		})
	})

	Describe("DELETE /desired_lrps/:process_guid", func() {
		var lrpRequest receptor.DesiredLRPCreateRequest
		var deleteErr error

		BeforeEach(func() {
			lrpRequest = newValidDesiredLRPCreateRequest()
			err := client.CreateDesiredLRP(lrpRequest)
			Ω(err).ShouldNot(HaveOccurred())

			deleteErr = client.DeleteDesiredLRP(lrpRequest.ProcessGuid)
		})

		It("responds without an error", func() {
			Ω(deleteErr).ShouldNot(HaveOccurred())
		})

		It("deletes the desired lrp with the matching process guid", func() {
			_, getErr := client.GetDesiredLRP(lrpRequest.ProcessGuid)
			Ω(getErr).Should(BeAssignableToTypeOf(receptor.Error{}))
			Ω(getErr.(receptor.Error).Type).Should(Equal(receptor.LRPNotFound))
		})
	})

	Describe("GET /desired_lrps", func() {
		var lrpResponses []receptor.DesiredLRPResponse
		const expectedLRPCount = 6
		var getErr error

		BeforeEach(func() {
			for i := 0; i < expectedLRPCount; i++ {
				err := client.CreateDesiredLRP(newValidDesiredLRPCreateRequest())
				Ω(err).ShouldNot(HaveOccurred())
			}
			lrpResponses, getErr = client.GetAllDesiredLRPs()
		})

		It("responds without an error", func() {
			Ω(getErr).ShouldNot(HaveOccurred())
		})

		It("fetches all of the desired lrps", func() {
			Ω(lrpResponses).Should(HaveLen(expectedLRPCount))
		})
	})

	Describe("GET /domains/:domain/desired_lrps", func() {
		const expectedDomain = "domain-1"
		const expectedLRPCount = 5
		var lrpResponses []receptor.DesiredLRPResponse
		var getErr error

		BeforeEach(func() {
			for i := 0; i < expectedLRPCount; i++ {
				lrp := newValidDesiredLRPCreateRequest()
				lrp.Domain = expectedDomain
				err := client.CreateDesiredLRP(lrp)
				Ω(err).ShouldNot(HaveOccurred())
			}
			for i := 0; i < expectedLRPCount; i++ {
				lrp := newValidDesiredLRPCreateRequest()
				lrp.Domain = "wrong-domain"
				err := client.CreateDesiredLRP(lrp)
				Ω(err).ShouldNot(HaveOccurred())
			}
			lrpResponses, getErr = client.GetAllDesiredLRPsByDomain(expectedDomain)
		})

		It("responds without an error", func() {
			Ω(getErr).ShouldNot(HaveOccurred())
		})

		It("fetches all of the desired lrps", func() {
			Ω(lrpResponses).Should(HaveLen(expectedLRPCount))
		})
	})
})

var processId int64

func newValidDesiredLRPCreateRequest() receptor.DesiredLRPCreateRequest {
	atomic.AddInt64(&processId, 1)

	return receptor.DesiredLRPCreateRequest{
		ProcessGuid: fmt.Sprintf("process-guid-%d", processId),
		Domain:      "test-domain",
		Stack:       "some-stack",
		Instances:   1,
		Actions: []models.ExecutorAction{
			{
				models.RunAction{
					Path: "/bin/bash",
				},
			},
		},
	}
}