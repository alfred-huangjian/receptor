package main_test

import (
	"fmt"
	"sync/atomic"

	"github.com/cloudfoundry-incubator/receptor"
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

	Describe("GET /desired_lrps", func() {
		var lrpRequests []receptor.DesiredLRPCreateRequest
		var lrpResponses []receptor.DesiredLRPResponse
		const expectedLRPcount = 6
		var getErr error

		BeforeEach(func() {
			lrpRequests = make([]receptor.DesiredLRPCreateRequest, expectedLRPcount)
			for i := 0; i < expectedLRPcount; i++ {
				lrpRequests[i] = newValidDesiredLRPCreateRequest()
				err := client.CreateDesiredLRP(lrpRequests[i])
				Ω(err).ShouldNot(HaveOccurred())
			}
			lrpResponses, getErr = client.GetAllDesiredLRPs()
		})

		It("responds without an error", func() {
			Ω(getErr).ShouldNot(HaveOccurred())
		})

		It("fetches all of the desired lrps", func() {
			Ω(lrpResponses).Should(HaveLen(expectedLRPcount))
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